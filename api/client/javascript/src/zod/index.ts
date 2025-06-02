/* eslint-disable no-useless-escape */
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { z as zod } from 'zod'

/**
 * List all add-ons.
 * @summary List add-ons
 */
export const listAddonsQueryIncludeDeletedDefault = false
export const listAddonsQueryIdItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listAddonsQueryKeyItemMax = 64

export const listAddonsQueryKeyItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const listAddonsQueryCurrencyItemMin = 3

export const listAddonsQueryCurrencyItemMax = 3

export const listAddonsQueryCurrencyItemRegExp = new RegExp('^[A-Z]{3}$')
export const listAddonsQueryPageDefault = 1
export const listAddonsQueryPageSizeDefault = 100
export const listAddonsQueryPageSizeMax = 1000

export const listAddonsQueryParams = zod.object({
  currency: zod
    .array(
      zod
        .string()
        .min(listAddonsQueryCurrencyItemMin)
        .max(listAddonsQueryCurrencyItemMax)
        .regex(listAddonsQueryCurrencyItemRegExp)
        .describe(
          'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
        )
    )
    .optional()
    .describe('Filter by addon.currency attribute'),
  id: zod
    .array(
      zod
        .string()
        .regex(listAddonsQueryIdItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by addon.id attribute'),
  includeDeleted: zod
    .boolean()
    .optional()
    .describe(
      'Include deleted add-ons in response.\n\nUsage: `?includeDeleted=true`'
    ),
  key: zod
    .array(
      zod
        .string()
        .min(1)
        .max(listAddonsQueryKeyItemMax)
        .regex(listAddonsQueryKeyItemRegExp)
        .describe(
          'A key is a unique string that is used to identify a resource.'
        )
    )
    .optional()
    .describe('Filter by addon.key attribute'),
  keyVersion: zod
    .record(zod.string(), zod.array(zod.number()))
    .optional()
    .describe('Filter by addon.key and addon.version attributes'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'key', 'version', 'created_at', 'updated_at'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listAddonsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listAddonsQueryPageSizeMax)
    .default(listAddonsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  status: zod
    .array(
      zod
        .enum(['draft', 'active', 'archived'])
        .describe(
          'The status of the add-on defined by the effectiveFrom and effectiveTo properties.'
        )
    )
    .optional()
    .describe(
      'Only return add-ons with the given status.\n\nUsage:\n- `?status=active`: return only the currently active add-ons\n- `?status=draft`: return only the draft add-ons\n- `?status=archived`: return only the archived add-ons'
    ),
})

/**
 * Create a new add-on.
 * @summary Create an add-on
 */
export const createAddonBodyNameMax = 256
export const createAddonBodyDescriptionMax = 1024
export const createAddonBodyKeyMax = 64

export const createAddonBodyKeyRegExp = new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createAddonBodyCurrencyMinOne = 3

export const createAddonBodyCurrencyMaxOne = 3

export const createAddonBodyCurrencyRegExpOne = new RegExp('^[A-Z]{3}$')
export const createAddonBodyCurrencyDefault = 'USD'
export const createAddonBodyRateCardsItemKeyMax = 64

export const createAddonBodyRateCardsItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createAddonBodyRateCardsItemNameMax = 256
export const createAddonBodyRateCardsItemDescriptionMax = 1024
export const createAddonBodyRateCardsItemFeatureKeyMax = 64

export const createAddonBodyRateCardsItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createAddonBodyRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const createAddonBodyRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const createAddonBodyRateCardsItemTaxConfigStripeCodeRegExp = new RegExp(
  '^txcd_\\d{8}$'
)
export const createAddonBodyRateCardsItemPriceAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const createAddonBodyRateCardsItemPricePaymentTermDefault = 'in_advance'
export const createAddonBodyRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemKeyMaxOne = 64

export const createAddonBodyRateCardsItemKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createAddonBodyRateCardsItemNameMaxOne = 256
export const createAddonBodyRateCardsItemDescriptionMaxOne = 1024
export const createAddonBodyRateCardsItemFeatureKeyMaxOne = 64

export const createAddonBodyRateCardsItemFeatureKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createAddonBodyRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const createAddonBodyRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const createAddonBodyRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const createAddonBodyRateCardsItemPriceAmountRegExpThree = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const createAddonBodyRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const createAddonBodyRateCardsItemPriceAmountRegExpFive = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const createAddonBodyRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMultiplierRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const createAddonBodyRateCardsItemPriceMultiplierDefault = '1'
export const createAddonBodyRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceAmountRegExpSeven = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const createAddonBodyRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createAddonBodyRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')

export const createAddonBody = zod
  .object({
    currency: zod
      .string()
      .min(createAddonBodyCurrencyMinOne)
      .max(createAddonBodyCurrencyMaxOne)
      .regex(createAddonBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .describe('The currency code of the add-on.'),
    description: zod
      .string()
      .max(createAddonBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    instanceType: zod
      .enum(['single', 'multiple'])
      .describe(
        'The instanceType of the add-on.\nSingle instance add-ons can be added to subscription only once while add-ons with multiple type can be added more then once.'
      )
      .describe(
        'The instanceType of the add-ons. Can be \"single\" or \"multiple\".'
      ),
    key: zod
      .string()
      .min(1)
      .max(createAddonBodyKeyMax)
      .regex(createAddonBodyKeyRegExp)
      .describe('A semi-unique identifier for the resource.'),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createAddonBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    rateCards: zod
      .array(
        zod
          .discriminatedUnion('type', [
            zod
              .object({
                billingCadence: zod
                  .string()
                  .nullable()
                  .describe(
                    'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                  ),
                description: zod
                  .string()
                  .max(createAddonBodyRateCardsItemDescriptionMax)
                  .optional()
                  .describe(
                    'Optional description of the resource. Maximum 1024 characters.'
                  ),
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('Percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        quantity: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemDiscountsUsageQuantityRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe(
                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                      )
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('Discount by type on a price')
                  .optional()
                  .describe(
                    'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                  ),
                entitlementTemplate: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        isSoftLimit: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                          ),
                        issueAfterReset: zod
                          .number()
                          .min(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMin
                          )
                          .optional()
                          .describe(
                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                          ),
                        issueAfterResetPriority: zod
                          .number()
                          .min(1)
                          .max(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                          )
                          .default(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                          )
                          .describe(
                            'Defines the grant priority for the default grant.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        preserveOverageAtReset: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                          ),
                        type: zod.enum(['metered']),
                        usagePeriod: zod
                          .string()
                          .optional()
                          .describe(
                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                          ),
                      })
                      .describe(
                        'The entitlement template with a metered entitlement.'
                      ),
                    zod
                      .object({
                        config: zod
                          .string()
                          .describe(
                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['static']),
                      })
                      .describe(
                        'Entitlement template of a static entitlement.'
                      ),
                    zod
                      .object({
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['boolean']),
                      })
                      .describe(
                        'Entitlement template of a boolean entitlement.'
                      ),
                  ])
                  .describe(
                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                  )
                  .optional()
                  .describe(
                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                  ),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemFeatureKeyMax)
                  .regex(createAddonBodyRateCardsItemFeatureKeyRegExp)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                key: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemKeyMax)
                  .regex(createAddonBodyRateCardsItemKeyRegExp)
                  .describe('A semi-unique identifier for the resource.'),
                metadata: zod
                  .record(zod.string(), zod.string())
                  .describe(
                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                  )
                  .nullish()
                  .describe('Additional metadata for the resource.'),
                name: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemNameMax)
                  .describe(
                    'Human-readable name for the resource. Between 1 and 256 characters.'
                  ),
                price: zod
                  .object({
                    amount: zod
                      .string()
                      .regex(createAddonBodyRateCardsItemPriceAmountRegExpOne)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the flat price.'),
                    paymentTerm: zod
                      .enum(['in_advance', 'in_arrears'])
                      .describe(
                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                      )
                      .default(
                        createAddonBodyRateCardsItemPricePaymentTermDefault
                      )
                      .describe(
                        'The payment term of the flat price.\nDefaults to in advance.'
                      ),
                    type: zod.enum(['flat']),
                  })
                  .describe('Flat price with payment term.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
                type: zod.enum(['flat_fee']),
              })
              .describe(
                'A flat fee rate card defines a one-time purchase or a recurring fee.'
              ),
            zod
              .object({
                billingCadence: zod
                  .string()
                  .describe('The billing cadence of the rate card.'),
                description: zod
                  .string()
                  .max(createAddonBodyRateCardsItemDescriptionMaxOne)
                  .optional()
                  .describe(
                    'Optional description of the resource. Maximum 1024 characters.'
                  ),
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('Percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        quantity: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemDiscountsUsageQuantityRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe(
                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                      )
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('Discount by type on a price')
                  .optional()
                  .describe(
                    'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                  ),
                entitlementTemplate: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        isSoftLimit: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                          ),
                        issueAfterReset: zod
                          .number()
                          .min(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                          )
                          .optional()
                          .describe(
                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                          ),
                        issueAfterResetPriority: zod
                          .number()
                          .min(1)
                          .max(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                          )
                          .default(
                            createAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                          )
                          .describe(
                            'Defines the grant priority for the default grant.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        preserveOverageAtReset: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                          ),
                        type: zod.enum(['metered']),
                        usagePeriod: zod
                          .string()
                          .optional()
                          .describe(
                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                          ),
                      })
                      .describe(
                        'The entitlement template with a metered entitlement.'
                      ),
                    zod
                      .object({
                        config: zod
                          .string()
                          .describe(
                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['static']),
                      })
                      .describe(
                        'Entitlement template of a static entitlement.'
                      ),
                    zod
                      .object({
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['boolean']),
                      })
                      .describe(
                        'Entitlement template of a boolean entitlement.'
                      ),
                  ])
                  .describe(
                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                  )
                  .optional()
                  .describe(
                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                  ),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemFeatureKeyMaxOne)
                  .regex(createAddonBodyRateCardsItemFeatureKeyRegExpOne)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                key: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemKeyMaxOne)
                  .regex(createAddonBodyRateCardsItemKeyRegExpOne)
                  .describe('A semi-unique identifier for the resource.'),
                metadata: zod
                  .record(zod.string(), zod.string())
                  .describe(
                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                  )
                  .nullish()
                  .describe('Additional metadata for the resource.'),
                name: zod
                  .string()
                  .min(1)
                  .max(createAddonBodyRateCardsItemNameMaxOne)
                  .describe(
                    'Human-readable name for the resource. Between 1 and 256 characters.'
                  ),
                price: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the flat price.'),
                        paymentTerm: zod
                          .enum(['in_advance', 'in_arrears'])
                          .describe(
                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                          )
                          .default(
                            createAddonBodyRateCardsItemPricePaymentTermDefaultTwo
                          )
                          .describe(
                            'The payment term of the flat price.\nDefaults to in advance.'
                          ),
                        type: zod.enum(['flat']),
                      })
                      .describe('Flat price with payment term.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the unit price.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMaximumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMinimumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        type: zod.enum(['unit']),
                      })
                      .describe('Unit price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMaximumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMinimumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        mode: zod
                          .enum(['volume', 'graduated'])
                          .describe('The mode of the tiered price.')
                          .describe(
                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                          ),
                        tiers: zod
                          .array(
                            zod
                              .object({
                                flatPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        createAddonBodyRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    type: zod
                                      .enum(['flat'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Flat price.')
                                  .nullable()
                                  .describe(
                                    'The flat price component of the tier.'
                                  ),
                                unitPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        createAddonBodyRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the unit price.'
                                      ),
                                    type: zod
                                      .enum(['unit'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Unit price.')
                                  .nullable()
                                  .describe(
                                    'The unit price component of the tier.'
                                  ),
                                upToAmount: zod
                                  .string()
                                  .regex(
                                    createAddonBodyRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                  ),
                              })
                              .describe(
                                'A price tier.\nAt least one price component is required in each tier.'
                              )
                          )
                          .min(1)
                          .describe(
                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                          ),
                        type: zod.enum(['tiered']),
                      })
                      .describe('Tiered price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMaximumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMinimumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        multiplier: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMultiplierRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .default(
                            createAddonBodyRateCardsItemPriceMultiplierDefault
                          )
                          .describe(
                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                          ),
                        type: zod.enum(['dynamic']),
                      })
                      .describe('Dynamic price with spend commitments.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The price of one package.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMaximumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceMinimumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        quantityPerPackage: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemPriceQuantityPerPackageRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The quantity per package.'),
                        type: zod.enum(['package']),
                      })
                      .describe('Package price with spend commitments.'),
                  ])
                  .describe('The price of the usage based rate card.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            createAddonBodyRateCardsItemTaxConfigStripeCodeRegExpOne
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
                type: zod.enum(['usage_based']),
              })
              .describe(
                'A usage-based rate card defines a price based on usage.'
              ),
          ])
          .describe(
            'A rate card defines the pricing and entitlement of a feature or service.'
          )
      )
      .describe('The rate cards of the add-on.'),
  })
  .describe('Resource create operation model.')

/**
 * Update add-on by id.
 * @summary Update add-on
 */
export const updateAddonPathAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateAddonParams = zod.object({
  addonId: zod.string().regex(updateAddonPathAddonIdRegExp),
})

export const updateAddonBodyNameMax = 256
export const updateAddonBodyDescriptionMax = 1024
export const updateAddonBodyRateCardsItemKeyMax = 64

export const updateAddonBodyRateCardsItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateAddonBodyRateCardsItemNameMax = 256
export const updateAddonBodyRateCardsItemDescriptionMax = 1024
export const updateAddonBodyRateCardsItemFeatureKeyMax = 64

export const updateAddonBodyRateCardsItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateAddonBodyRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const updateAddonBodyRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const updateAddonBodyRateCardsItemTaxConfigStripeCodeRegExp = new RegExp(
  '^txcd_\\d{8}$'
)
export const updateAddonBodyRateCardsItemPriceAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateAddonBodyRateCardsItemPricePaymentTermDefault = 'in_advance'
export const updateAddonBodyRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemKeyMaxOne = 64

export const updateAddonBodyRateCardsItemKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateAddonBodyRateCardsItemNameMaxOne = 256
export const updateAddonBodyRateCardsItemDescriptionMaxOne = 1024
export const updateAddonBodyRateCardsItemFeatureKeyMaxOne = 64

export const updateAddonBodyRateCardsItemFeatureKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateAddonBodyRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const updateAddonBodyRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const updateAddonBodyRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const updateAddonBodyRateCardsItemPriceAmountRegExpThree = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateAddonBodyRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const updateAddonBodyRateCardsItemPriceAmountRegExpFive = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateAddonBodyRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMultiplierRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateAddonBodyRateCardsItemPriceMultiplierDefault = '1'
export const updateAddonBodyRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceAmountRegExpSeven = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateAddonBodyRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateAddonBodyRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')

export const updateAddonBody = zod
  .object({
    description: zod
      .string()
      .max(updateAddonBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    instanceType: zod
      .enum(['single', 'multiple'])
      .describe(
        'The instanceType of the add-on.\nSingle instance add-ons can be added to subscription only once while add-ons with multiple type can be added more then once.'
      )
      .describe(
        'The instanceType of the add-ons. Can be \"single\" or \"multiple\".'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updateAddonBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    rateCards: zod
      .array(
        zod
          .discriminatedUnion('type', [
            zod
              .object({
                billingCadence: zod
                  .string()
                  .nullable()
                  .describe(
                    'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                  ),
                description: zod
                  .string()
                  .max(updateAddonBodyRateCardsItemDescriptionMax)
                  .optional()
                  .describe(
                    'Optional description of the resource. Maximum 1024 characters.'
                  ),
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('Percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        quantity: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemDiscountsUsageQuantityRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe(
                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                      )
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('Discount by type on a price')
                  .optional()
                  .describe(
                    'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                  ),
                entitlementTemplate: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        isSoftLimit: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                          ),
                        issueAfterReset: zod
                          .number()
                          .min(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMin
                          )
                          .optional()
                          .describe(
                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                          ),
                        issueAfterResetPriority: zod
                          .number()
                          .min(1)
                          .max(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                          )
                          .default(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                          )
                          .describe(
                            'Defines the grant priority for the default grant.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        preserveOverageAtReset: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                          ),
                        type: zod.enum(['metered']),
                        usagePeriod: zod
                          .string()
                          .optional()
                          .describe(
                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                          ),
                      })
                      .describe(
                        'The entitlement template with a metered entitlement.'
                      ),
                    zod
                      .object({
                        config: zod
                          .string()
                          .describe(
                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['static']),
                      })
                      .describe(
                        'Entitlement template of a static entitlement.'
                      ),
                    zod
                      .object({
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['boolean']),
                      })
                      .describe(
                        'Entitlement template of a boolean entitlement.'
                      ),
                  ])
                  .describe(
                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                  )
                  .optional()
                  .describe(
                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                  ),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemFeatureKeyMax)
                  .regex(updateAddonBodyRateCardsItemFeatureKeyRegExp)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                key: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemKeyMax)
                  .regex(updateAddonBodyRateCardsItemKeyRegExp)
                  .describe('A semi-unique identifier for the resource.'),
                metadata: zod
                  .record(zod.string(), zod.string())
                  .describe(
                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                  )
                  .nullish()
                  .describe('Additional metadata for the resource.'),
                name: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemNameMax)
                  .describe(
                    'Human-readable name for the resource. Between 1 and 256 characters.'
                  ),
                price: zod
                  .object({
                    amount: zod
                      .string()
                      .regex(updateAddonBodyRateCardsItemPriceAmountRegExpOne)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the flat price.'),
                    paymentTerm: zod
                      .enum(['in_advance', 'in_arrears'])
                      .describe(
                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                      )
                      .default(
                        updateAddonBodyRateCardsItemPricePaymentTermDefault
                      )
                      .describe(
                        'The payment term of the flat price.\nDefaults to in advance.'
                      ),
                    type: zod.enum(['flat']),
                  })
                  .describe('Flat price with payment term.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
                type: zod.enum(['flat_fee']),
              })
              .describe(
                'A flat fee rate card defines a one-time purchase or a recurring fee.'
              ),
            zod
              .object({
                billingCadence: zod
                  .string()
                  .describe('The billing cadence of the rate card.'),
                description: zod
                  .string()
                  .max(updateAddonBodyRateCardsItemDescriptionMaxOne)
                  .optional()
                  .describe(
                    'Optional description of the resource. Maximum 1024 characters.'
                  ),
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('Percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        quantity: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemDiscountsUsageQuantityRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe(
                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                      )
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('Discount by type on a price')
                  .optional()
                  .describe(
                    'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                  ),
                entitlementTemplate: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        isSoftLimit: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                          ),
                        issueAfterReset: zod
                          .number()
                          .min(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                          )
                          .optional()
                          .describe(
                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                          ),
                        issueAfterResetPriority: zod
                          .number()
                          .min(1)
                          .max(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                          )
                          .default(
                            updateAddonBodyRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                          )
                          .describe(
                            'Defines the grant priority for the default grant.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        preserveOverageAtReset: zod
                          .boolean()
                          .optional()
                          .describe(
                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                          ),
                        type: zod.enum(['metered']),
                        usagePeriod: zod
                          .string()
                          .optional()
                          .describe(
                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                          ),
                      })
                      .describe(
                        'The entitlement template with a metered entitlement.'
                      ),
                    zod
                      .object({
                        config: zod
                          .string()
                          .describe(
                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['static']),
                      })
                      .describe(
                        'Entitlement template of a static entitlement.'
                      ),
                    zod
                      .object({
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .optional()
                          .describe('Additional metadata for the feature.'),
                        type: zod.enum(['boolean']),
                      })
                      .describe(
                        'Entitlement template of a boolean entitlement.'
                      ),
                  ])
                  .describe(
                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                  )
                  .optional()
                  .describe(
                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                  ),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemFeatureKeyMaxOne)
                  .regex(updateAddonBodyRateCardsItemFeatureKeyRegExpOne)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                key: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemKeyMaxOne)
                  .regex(updateAddonBodyRateCardsItemKeyRegExpOne)
                  .describe('A semi-unique identifier for the resource.'),
                metadata: zod
                  .record(zod.string(), zod.string())
                  .describe(
                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                  )
                  .nullish()
                  .describe('Additional metadata for the resource.'),
                name: zod
                  .string()
                  .min(1)
                  .max(updateAddonBodyRateCardsItemNameMaxOne)
                  .describe(
                    'Human-readable name for the resource. Between 1 and 256 characters.'
                  ),
                price: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the flat price.'),
                        paymentTerm: zod
                          .enum(['in_advance', 'in_arrears'])
                          .describe(
                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                          )
                          .default(
                            updateAddonBodyRateCardsItemPricePaymentTermDefaultTwo
                          )
                          .describe(
                            'The payment term of the flat price.\nDefaults to in advance.'
                          ),
                        type: zod.enum(['flat']),
                      })
                      .describe('Flat price with payment term.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the unit price.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMaximumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMinimumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        type: zod.enum(['unit']),
                      })
                      .describe('Unit price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMaximumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMinimumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        mode: zod
                          .enum(['volume', 'graduated'])
                          .describe('The mode of the tiered price.')
                          .describe(
                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                          ),
                        tiers: zod
                          .array(
                            zod
                              .object({
                                flatPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        updateAddonBodyRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    type: zod
                                      .enum(['flat'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Flat price.')
                                  .nullable()
                                  .describe(
                                    'The flat price component of the tier.'
                                  ),
                                unitPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        updateAddonBodyRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the unit price.'
                                      ),
                                    type: zod
                                      .enum(['unit'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Unit price.')
                                  .nullable()
                                  .describe(
                                    'The unit price component of the tier.'
                                  ),
                                upToAmount: zod
                                  .string()
                                  .regex(
                                    updateAddonBodyRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                  ),
                              })
                              .describe(
                                'A price tier.\nAt least one price component is required in each tier.'
                              )
                          )
                          .min(1)
                          .describe(
                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                          ),
                        type: zod.enum(['tiered']),
                      })
                      .describe('Tiered price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMaximumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMinimumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        multiplier: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMultiplierRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .default(
                            updateAddonBodyRateCardsItemPriceMultiplierDefault
                          )
                          .describe(
                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                          ),
                        type: zod.enum(['dynamic']),
                      })
                      .describe('Dynamic price with spend commitments.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The price of one package.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMaximumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceMinimumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        quantityPerPackage: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemPriceQuantityPerPackageRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The quantity per package.'),
                        type: zod.enum(['package']),
                      })
                      .describe('Package price with spend commitments.'),
                  ])
                  .describe('The price of the usage based rate card.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            updateAddonBodyRateCardsItemTaxConfigStripeCodeRegExpOne
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
                type: zod.enum(['usage_based']),
              })
              .describe(
                'A usage-based rate card defines a price based on usage.'
              ),
          ])
          .describe(
            'A rate card defines the pricing and entitlement of a feature or service.'
          )
      )
      .describe('The rate cards of the add-on.'),
  })
  .describe('Resource update operation model.')

/**
 * Get add-on by id or key. The latest published version is returned if latter is used.
 * @summary Get add-on
 */
export const getAddonPathAddonIdMax = 64

export const getAddonPathAddonIdRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getAddonParams = zod.object({
  addonId: zod
    .string()
    .min(1)
    .max(getAddonPathAddonIdMax)
    .regex(getAddonPathAddonIdRegExp),
})

export const getAddonQueryIncludeLatestDefault = false

export const getAddonQueryParams = zod.object({
  includeLatest: zod
    .boolean()
    .optional()
    .describe(
      'Include latest version of the add-on instead of the version in active state.\n\nUsage: `?includeLatest=true`'
    ),
})

/**
 * Soft delete add-on by id.

Once a add-on is deleted it cannot be undeleted.
 * @summary Delete add-on
 */
export const deleteAddonPathAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteAddonParams = zod.object({
  addonId: zod.string().regex(deleteAddonPathAddonIdRegExp),
})

/**
 * Archive a add-on version.
 * @summary Archive add-on version
 */
export const archiveAddonPathAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const archiveAddonParams = zod.object({
  addonId: zod.string().regex(archiveAddonPathAddonIdRegExp),
})

/**
 * Publish a add-on version.
 * @summary Publish add-on
 */
export const publishAddonPathAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const publishAddonParams = zod.object({
  addonId: zod.string().regex(publishAddonPathAddonIdRegExp),
})

/**
 * List apps.
 * @summary List apps
 */
export const listAppsQueryPageDefault = 1
export const listAppsQueryPageSizeDefault = 100
export const listAppsQueryPageSizeMax = 1000

export const listAppsQueryParams = zod.object({
  page: zod
    .number()
    .min(1)
    .default(listAppsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listAppsQueryPageSizeMax)
    .default(listAppsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * @summary Submit draft synchronization results
 */
export const appCustomInvoicingDraftSynchronizedPathInvoiceIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const appCustomInvoicingDraftSynchronizedParams = zod.object({
  invoiceId: zod
    .string()
    .regex(appCustomInvoicingDraftSynchronizedPathInvoiceIdRegExp),
})

export const appCustomInvoicingDraftSynchronizedBodyInvoicingInvoiceNumberMaxOne = 256
export const appCustomInvoicingDraftSynchronizedBodyInvoicingLineExternalIdsItemLineIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const appCustomInvoicingDraftSynchronizedBodyInvoicingLineDiscountExternalIdsItemLineDiscountIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const appCustomInvoicingDraftSynchronizedBody = zod
  .object({
    invoicing: zod
      .object({
        externalId: zod
          .string()
          .optional()
          .describe(
            "If set the invoice's invoicing external ID will be set to this value."
          ),
        invoiceNumber: zod
          .string()
          .min(1)
          .max(
            appCustomInvoicingDraftSynchronizedBodyInvoicingInvoiceNumberMaxOne
          )
          .describe(
            'InvoiceNumber is a unique identifier for the invoice, generated by the\ninvoicing app.\n\nThe uniqueness depends on a lot of factors:\n- app setting (unique per app or unique per customer)\n- multiple app scenarios (multiple apps generating invoices with the same prefix)'
          )
          .optional()
          .describe("If set the invoice's number will be set to this value."),
        lineDiscountExternalIds: zod
          .array(
            zod
              .object({
                externalId: zod
                  .string()
                  .describe(
                    "The external ID (e.g. custom invoicing system's ID)."
                  ),
                lineDiscountId: zod
                  .string()
                  .regex(
                    appCustomInvoicingDraftSynchronizedBodyInvoicingLineDiscountExternalIdsItemLineDiscountIdRegExp
                  )
                  .describe('The line discount ID.'),
              })
              .describe('Mapping between line discounts and external IDs.')
          )
          .optional()
          .describe(
            "If set the invoice's line discount external IDs will be set to this value.\n\nThis can be used to reference the external system's entities in the\ninvoice."
          ),
        lineExternalIds: zod
          .array(
            zod
              .object({
                externalId: zod
                  .string()
                  .describe(
                    "The external ID (e.g. custom invoicing system's ID)."
                  ),
                lineId: zod
                  .string()
                  .regex(
                    appCustomInvoicingDraftSynchronizedBodyInvoicingLineExternalIdsItemLineIdRegExp
                  )
                  .describe('The line ID.'),
              })
              .describe('Mapping between lines and external IDs.')
          )
          .optional()
          .describe(
            "If set the invoice's line external IDs will be set to this value.\n\nThis can be used to reference the external system's entities in the\ninvoice."
          ),
      })
      .describe(
        "Information to synchronize the invoice.\n\nCan be used to store external app's IDs on the invoice or lines."
      )
      .optional()
      .describe('The result of the synchronization.'),
  })
  .describe('Information to finalize the draft details of an invoice.')

/**
 * @summary Submit issuing synchronization results
 */
export const appCustomInvoicingIssuingSynchronizedPathInvoiceIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const appCustomInvoicingIssuingSynchronizedParams = zod.object({
  invoiceId: zod
    .string()
    .regex(appCustomInvoicingIssuingSynchronizedPathInvoiceIdRegExp),
})

export const appCustomInvoicingIssuingSynchronizedBodyInvoicingInvoiceNumberMaxOne = 256

export const appCustomInvoicingIssuingSynchronizedBody = zod
  .object({
    invoicing: zod
      .object({
        invoiceNumber: zod
          .string()
          .min(1)
          .max(
            appCustomInvoicingIssuingSynchronizedBodyInvoicingInvoiceNumberMaxOne
          )
          .describe(
            'InvoiceNumber is a unique identifier for the invoice, generated by the\ninvoicing app.\n\nThe uniqueness depends on a lot of factors:\n- app setting (unique per app or unique per customer)\n- multiple app scenarios (multiple apps generating invoices with the same prefix)'
          )
          .optional()
          .describe("If set the invoice's number will be set to this value."),
        sentToCustomerAt: zod
          .date()
          .optional()
          .describe(
            "If set the invoice's sent to customer at will be set to this value."
          ),
      })
      .describe('Information to finalize the invoicing details of an invoice.')
      .optional()
      .describe('The result of the synchronization.'),
    payment: zod
      .object({
        externalId: zod
          .string()
          .optional()
          .describe(
            "If set the invoice's payment external ID will be set to this value."
          ),
      })
      .describe('Information to finalize the payment details of an invoice.')
      .optional()
      .describe('The result of the payment synchronization.'),
  })
  .describe(
    'Information to finalize the invoice.\n\nIf invoicing.invoiceNumber is not set, then a new invoice number will be generated (INV- prefix).'
  )

/**
 * @summary Update payment status
 */
export const appCustomInvoicingUpdatePaymentStatusPathInvoiceIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const appCustomInvoicingUpdatePaymentStatusParams = zod.object({
  invoiceId: zod
    .string()
    .regex(appCustomInvoicingUpdatePaymentStatusPathInvoiceIdRegExp),
})

export const appCustomInvoicingUpdatePaymentStatusBody = zod
  .object({
    trigger: zod
      .enum([
        'paid',
        'payment_failed',
        'payment_uncollectible',
        'payment_overdue',
        'action_required',
        'void',
      ])
      .describe('Payment trigger to execute on a finalized invoice.')
      .describe('The trigger to be executed on the invoice.'),
  })
  .describe(
    "Update payment status request.\n\nCan be used to manipulate invoice's payment status (when custominvoicing app is being used)."
  )

/**
 * Get the app.
 * @summary Get app
 */
export const getAppPathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getAppParams = zod.object({
  id: zod.string().regex(getAppPathIdRegExp),
})

/**
 * Update an app.
 * @summary Update app
 */
export const updateAppPathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateAppParams = zod.object({
  id: zod.string().regex(updateAppPathIdRegExp),
})

export const updateAppBodyNameMax = 256
export const updateAppBodyDescriptionMax = 1024
export const updateAppBodyNameMaxOne = 256
export const updateAppBodyDescriptionMaxOne = 1024
export const updateAppBodyNameMaxTwo = 256
export const updateAppBodyDescriptionMaxTwo = 1024

export const updateAppBody = zod
  .discriminatedUnion('type', [
    zod
      .object({
        default: zod
          .boolean()
          .describe(
            'Default for the app type\nOnly one app of each type can be default.'
          ),
        description: zod
          .string()
          .max(updateAppBodyDescriptionMax)
          .optional()
          .describe(
            'Optional description of the resource. Maximum 1024 characters.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .nullish()
          .describe('Additional metadata for the resource.'),
        name: zod
          .string()
          .min(1)
          .max(updateAppBodyNameMax)
          .describe(
            'Human-readable name for the resource. Between 1 and 256 characters.'
          ),
        secretAPIKey: zod.string().optional().describe('The Stripe API key.'),
        type: zod.enum(['stripe']),
      })
      .describe('Resource update operation model.'),
    zod
      .object({
        default: zod
          .boolean()
          .describe(
            'Default for the app type\nOnly one app of each type can be default.'
          ),
        description: zod
          .string()
          .max(updateAppBodyDescriptionMaxOne)
          .optional()
          .describe(
            'Optional description of the resource. Maximum 1024 characters.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .nullish()
          .describe('Additional metadata for the resource.'),
        name: zod
          .string()
          .min(1)
          .max(updateAppBodyNameMaxOne)
          .describe(
            'Human-readable name for the resource. Between 1 and 256 characters.'
          ),
        type: zod.enum(['sandbox']),
      })
      .describe('Resource update operation model.'),
    zod
      .object({
        default: zod
          .boolean()
          .describe(
            'Default for the app type\nOnly one app of each type can be default.'
          ),
        description: zod
          .string()
          .max(updateAppBodyDescriptionMaxTwo)
          .optional()
          .describe(
            'Optional description of the resource. Maximum 1024 characters.'
          ),
        enableDraftSyncHook: zod
          .boolean()
          .describe(
            'Enable draft.sync hook.\n\nIf the hook is not enabled, the invoice will be progressed to the next state automatically.'
          ),
        enableIssuingSyncHook: zod
          .boolean()
          .describe(
            'Enable issuing.sync hook.\n\nIf the hook is not enabled, the invoice will be progressed to the next state automatically.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .nullish()
          .describe('Additional metadata for the resource.'),
        name: zod
          .string()
          .min(1)
          .max(updateAppBodyNameMaxTwo)
          .describe(
            'Human-readable name for the resource. Between 1 and 256 characters.'
          ),
        type: zod.enum(['custom_invoicing']),
      })
      .describe('Resource update operation model.'),
  ])
  .describe('App ReplaceUpdate Model')

/**
 * Uninstall an app.
 * @summary Uninstall app
 */
export const uninstallAppPathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const uninstallAppParams = zod.object({
  id: zod.string().regex(uninstallAppPathIdRegExp),
})

/**
 * Update the Stripe API key.
 * @deprecated
 * @summary Update Stripe API key
 */
export const updateStripeAPIKeyPathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateStripeAPIKeyParams = zod.object({
  id: zod.string().regex(updateStripeAPIKeyPathIdRegExp),
})

export const updateStripeAPIKeyBody = zod
  .object({
    secretAPIKey: zod.string(),
  })
  .describe(
    'The Stripe API key input.\nUsed to authenticate with the Stripe API.'
  )

/**
 * Handle stripe webhooks for apps.
 * @summary Stripe webhook
 */
export const appStripeWebhookPathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const appStripeWebhookParams = zod.object({
  id: zod.string().regex(appStripeWebhookPathIdRegExp),
})

export const appStripeWebhookBody = zod
  .object({
    created: zod.number().describe('The event created timestamp.'),
    data: zod
      .object({
        object: zod.any(),
      })
      .describe('The event data.'),
    id: zod.string().describe('The event ID.'),
    livemode: zod.boolean().describe('Live mode.'),
    type: zod.string().describe('The event type.'),
  })
  .describe('Stripe webhook event.')

/**
 * List customer overrides using the specified filters.

The response will include the customer override values and the merged billing profile values.

If the includeAllCustomers is set to true, the list contains all customers. This mode is
useful for getting the current effective billing workflow settings for all users regardless
if they have customer orverrides or not.
 * @summary List customer overrides
 */
export const listBillingProfileCustomerOverridesQueryBillingProfileItemRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const listBillingProfileCustomerOverridesQueryIncludeAllCustomersDefault =
  true
export const listBillingProfileCustomerOverridesQueryCustomerIdItemRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const listBillingProfileCustomerOverridesQueryPageDefault = 1
export const listBillingProfileCustomerOverridesQueryPageSizeDefault = 100
export const listBillingProfileCustomerOverridesQueryPageSizeMax = 1000

export const listBillingProfileCustomerOverridesQueryParams = zod.object({
  billingProfile: zod
    .array(
      zod
        .string()
        .regex(listBillingProfileCustomerOverridesQueryBillingProfileItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by billing profile.'),
  customerId: zod
    .array(
      zod
        .string()
        .regex(listBillingProfileCustomerOverridesQueryCustomerIdItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by customer id.'),
  customerKey: zod.string().optional().describe('Filter by customer key'),
  customerName: zod.string().optional().describe('Filter by customer name.'),
  customerPrimaryEmail: zod
    .string()
    .optional()
    .describe('Filter by customer primary email'),
  expand: zod
    .array(
      zod
        .enum(['apps', 'customer'])
        .describe(
          'CustomerOverrideExpand specifies the parts of the profile to expand.'
        )
    )
    .optional()
    .describe('Expand the response with additional details.'),
  includeAllCustomers: zod
    .boolean()
    .default(listBillingProfileCustomerOverridesQueryIncludeAllCustomersDefault)
    .describe(
      'Include customers without customer overrides.\n\nIf set to false only the customers specifically associated with a billing profile will be returned.\n\nIf set to true, in case of the default billing profile, all customers will be returned.'
    ),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum([
      'customerId',
      'customerName',
      'customerKey',
      'customerPrimaryEmail',
      'customerCreatedAt',
    ])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listBillingProfileCustomerOverridesQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listBillingProfileCustomerOverridesQueryPageSizeMax)
    .default(listBillingProfileCustomerOverridesQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * The customer override can be used to pin a given customer to a billing profile
different from the default one.

This can be used to test the effect of different billing profiles before making them
the default ones or have different workflow settings for example for enterprise customers.
 * @summary Create a new or update a customer override
 */
export const upsertBillingProfileCustomerOverridePathCustomerIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const upsertBillingProfileCustomerOverrideParams = zod.object({
  customerId: zod
    .string()
    .regex(upsertBillingProfileCustomerOverridePathCustomerIdRegExp),
})

export const upsertBillingProfileCustomerOverrideBodyBillingProfileIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const upsertBillingProfileCustomerOverrideBody = zod
  .object({
    billingProfileId: zod
      .string()
      .regex(upsertBillingProfileCustomerOverrideBodyBillingProfileIdRegExp)
      .optional()
      .describe(
        'The billing profile this override is associated with.\n\nIf not provided, the default billing profile is chosen if available.'
      ),
  })
  .describe(
    'Payload for creating a new or updating an existing customer override.'
  )

/**
 * Get a customer override by customer id.

The response will include the customer override values and the merged billing profile values.

If the customer override is not found, the default billing profile's values are returned. This behavior
allows for getting a merged profile regardless of the customer override existence.
 * @summary Get a customer override
 */
export const getBillingProfileCustomerOverridePathCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getBillingProfileCustomerOverrideParams = zod.object({
  customerId: zod
    .string()
    .regex(getBillingProfileCustomerOverridePathCustomerIdRegExp),
})

export const getBillingProfileCustomerOverrideQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['apps', 'customer'])
        .describe(
          'CustomerOverrideExpand specifies the parts of the profile to expand.'
        )
    )
    .optional(),
})

/**
 * Delete a customer override by customer id.

This will remove the customer override and the customer will be subject to the default
billing profile's settings again.
 * @summary Delete a customer override
 */
export const deleteBillingProfileCustomerOverridePathCustomerIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const deleteBillingProfileCustomerOverrideParams = zod.object({
  customerId: zod
    .string()
    .regex(deleteBillingProfileCustomerOverridePathCustomerIdRegExp),
})

/**
 * Create a new pending line item (charge).

This call is used to create a new pending line item for the customer if required a new
gathering invoice will be created.

A new invoice will be created if:
- there is no invoice in gathering state
- the currency of the line item doesn't match the currency of any invoices in gathering state
 * @summary Create pending line items
 */
export const createPendingInvoiceLinePathCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createPendingInvoiceLineParams = zod.object({
  customerId: zod.string().regex(createPendingInvoiceLinePathCustomerIdRegExp),
})

export const createPendingInvoiceLineBodyCurrencyMinOne = 3

export const createPendingInvoiceLineBodyCurrencyMaxOne = 3

export const createPendingInvoiceLineBodyCurrencyRegExpOne = new RegExp(
  '^[A-Z]{3}$'
)
export const createPendingInvoiceLineBodyLinesItemNameMax = 256
export const createPendingInvoiceLineBodyLinesItemDescriptionMax = 1024
export const createPendingInvoiceLineBodyLinesItemTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const createPendingInvoiceLineBodyLinesItemPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPricePaymentTermDefault =
  'in_advance'
export const createPendingInvoiceLineBodyLinesItemPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMultiplierDefault = '1'
export const createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemFeatureKeyMax = 64

export const createPendingInvoiceLineBodyLinesItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createPendingInvoiceLineBodyLinesItemRateCardFeatureKeyMax = 64

export const createPendingInvoiceLineBodyLinesItemRateCardFeatureKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createPendingInvoiceLineBodyLinesItemRateCardTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPricePaymentTermDefault =
  'in_advance'
export const createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMultiplierDefault =
  '1'
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const createPendingInvoiceLineBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPendingInvoiceLineBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')

export const createPendingInvoiceLineBody = zod
  .object({
    currency: zod
      .string()
      .min(createPendingInvoiceLineBodyCurrencyMinOne)
      .max(createPendingInvoiceLineBodyCurrencyMaxOne)
      .regex(createPendingInvoiceLineBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .describe('The currency of the lines to be created.'),
    lines: zod
      .array(
        zod
          .object({
            description: zod
              .string()
              .max(createPendingInvoiceLineBodyLinesItemDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            featureKey: zod
              .string()
              .min(1)
              .max(createPendingInvoiceLineBodyLinesItemFeatureKeyMax)
              .regex(createPendingInvoiceLineBodyLinesItemFeatureKeyRegExp)
              .optional()
              .describe('The feature that the usage is based on.'),
            invoiceAt: zod
              .date()
              .describe('The time this line item should be invoiced.'),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(createPendingInvoiceLineBodyLinesItemNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            period: zod
              .object({
                from: zod.date().describe('Period start time.'),
                to: zod.date().describe('Period end time.'),
              })
              .describe('A period with a start and end time.')
              .describe(
                'Period of the line item applies to for revenue recognition pruposes.\n\nBilling always treats periods as start being inclusive and end being exclusive.'
              ),
            price: zod
              .discriminatedUnion('type', [
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the flat price.'),
                    paymentTerm: zod
                      .enum(['in_advance', 'in_arrears'])
                      .describe(
                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                      )
                      .default(
                        createPendingInvoiceLineBodyLinesItemPricePaymentTermDefault
                      )
                      .describe(
                        'The payment term of the flat price.\nDefaults to in advance.'
                      ),
                    type: zod.enum(['flat']),
                  })
                  .describe('Flat price with payment term.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the unit price.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    type: zod.enum(['unit']),
                  })
                  .describe('Unit price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    mode: zod
                      .enum(['volume', 'graduated'])
                      .describe('The mode of the tiered price.')
                      .describe(
                        'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                      ),
                    tiers: zod
                      .array(
                        zod
                          .object({
                            flatPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    createPendingInvoiceLineBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                type: zod
                                  .enum(['flat'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Flat price.')
                              .nullable()
                              .describe(
                                'The flat price component of the tier.'
                              ),
                            unitPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    createPendingInvoiceLineBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                type: zod
                                  .enum(['unit'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Unit price.')
                              .nullable()
                              .describe(
                                'The unit price component of the tier.'
                              ),
                            upToAmount: zod
                              .string()
                              .regex(
                                createPendingInvoiceLineBodyLinesItemPriceTiersItemUpToAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .optional()
                              .describe(
                                'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                              ),
                          })
                          .describe(
                            'A price tier.\nAt least one price component is required in each tier.'
                          )
                      )
                      .min(1)
                      .describe(
                        'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                      ),
                    type: zod.enum(['tiered']),
                  })
                  .describe('Tiered price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    multiplier: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMultiplierRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .default(
                        createPendingInvoiceLineBodyLinesItemPriceMultiplierDefault
                      )
                      .describe(
                        'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                      ),
                    type: zod.enum(['dynamic']),
                  })
                  .describe('Dynamic price with spend commitments.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The price of one package.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMaximumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceMinimumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    quantityPerPackage: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemPriceQuantityPerPackageRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The quantity per package.'),
                    type: zod.enum(['package']),
                  })
                  .describe('Package price with spend commitments.'),
              ])
              .describe('The price of the usage based rate card.')
              .optional()
              .describe('Price of the usage-based item being sold.'),
            rateCard: zod
              .object({
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('A percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        quantity: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe('A usage discount.')
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('A discount by type.')
                  .optional()
                  .describe('The discounts that are applied to the line.'),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(
                    createPendingInvoiceLineBodyLinesItemRateCardFeatureKeyMax
                  )
                  .regex(
                    createPendingInvoiceLineBodyLinesItemRateCardFeatureKeyRegExp
                  )
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                price: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the flat price.'),
                        paymentTerm: zod
                          .enum(['in_advance', 'in_arrears'])
                          .describe(
                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                          )
                          .default(
                            createPendingInvoiceLineBodyLinesItemRateCardPricePaymentTermDefault
                          )
                          .describe(
                            'The payment term of the flat price.\nDefaults to in advance.'
                          ),
                        type: zod.enum(['flat']),
                      })
                      .describe('Flat price with payment term.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the unit price.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        type: zod.enum(['unit']),
                      })
                      .describe('Unit price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        mode: zod
                          .enum(['volume', 'graduated'])
                          .describe('The mode of the tiered price.')
                          .describe(
                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                          ),
                        tiers: zod
                          .array(
                            zod
                              .object({
                                flatPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    type: zod
                                      .enum(['flat'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Flat price.')
                                  .nullable()
                                  .describe(
                                    'The flat price component of the tier.'
                                  ),
                                unitPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the unit price.'
                                      ),
                                    type: zod
                                      .enum(['unit'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Unit price.')
                                  .nullable()
                                  .describe(
                                    'The unit price component of the tier.'
                                  ),
                                upToAmount: zod
                                  .string()
                                  .regex(
                                    createPendingInvoiceLineBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                  ),
                              })
                              .describe(
                                'A price tier.\nAt least one price component is required in each tier.'
                              )
                          )
                          .min(1)
                          .describe(
                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                          ),
                        type: zod.enum(['tiered']),
                      })
                      .describe('Tiered price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        multiplier: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMultiplierRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .default(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMultiplierDefault
                          )
                          .describe(
                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                          ),
                        type: zod.enum(['dynamic']),
                      })
                      .describe('Dynamic price with spend commitments.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The price of one package.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMaximumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceMinimumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        quantityPerPackage: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The quantity per package.'),
                        type: zod.enum(['package']),
                      })
                      .describe('Package price with spend commitments.'),
                  ])
                  .describe('The price of the usage based rate card.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            createPendingInvoiceLineBodyLinesItemRateCardTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
              })
              .describe(
                'InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line.'
              )
              .optional()
              .describe(
                'The rate card that is used for this line.\n\nThe rate card captures the intent of the price and discounts for the usage-based item.'
              ),
            taxConfig: zod
              .object({
                behavior: zod
                  .enum(['inclusive', 'exclusive'])
                  .describe(
                    'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                  )
                  .optional()
                  .describe(
                    "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                  ),
                customInvoicing: zod
                  .object({
                    code: zod
                      .string()
                      .describe(
                        'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                      ),
                  })
                  .describe('Custom invoicing tax config.')
                  .optional()
                  .describe('Custom invoicing tax config.'),
                stripe: zod
                  .object({
                    code: zod
                      .string()
                      .regex(
                        createPendingInvoiceLineBodyLinesItemTaxConfigStripeCodeRegExp
                      )
                      .describe(
                        'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                      ),
                  })
                  .describe('The tax config for Stripe.')
                  .optional()
                  .describe('Stripe tax config.'),
              })
              .describe('Set of provider specific tax configs.')
              .optional()
              .describe(
                'Tax config specify the tax configuration for this line.'
              ),
          })
          .describe(
            'InvoicePendingLineCreate represents the create model for an invoice line that is sold to the customer based on usage.'
          )
      )
      .min(1)
      .describe('The lines to be created.'),
  })
  .describe(
    'InvoicePendingLineCreate represents the create model for a pending invoice line.'
  )

/**
 * Simulate an invoice for a customer.

This call will simulate an invoice for a customer based on the pending line items.

The call will return the total amount of the invoice and the line items that will be included in the invoice.
 * @summary Simulate an invoice for a customer
 */
export const simulateInvoicePathCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const simulateInvoiceParams = zod.object({
  customerId: zod.string().regex(simulateInvoicePathCustomerIdRegExp),
})

export const simulateInvoiceBodyNumberMaxOne = 256
export const simulateInvoiceBodyCurrencyMinOne = 3

export const simulateInvoiceBodyCurrencyMaxOne = 3

export const simulateInvoiceBodyCurrencyRegExpOne = new RegExp('^[A-Z]{3}$')
export const simulateInvoiceBodyLinesItemNameMax = 256
export const simulateInvoiceBodyLinesItemDescriptionMax = 1024
export const simulateInvoiceBodyLinesItemTaxConfigStripeCodeRegExp = new RegExp(
  '^txcd_\\d{8}$'
)
export const simulateInvoiceBodyLinesItemPriceAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const simulateInvoiceBodyLinesItemPricePaymentTermDefault = 'in_advance'
export const simulateInvoiceBodyLinesItemPriceAmountRegExpThree = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMultiplierRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const simulateInvoiceBodyLinesItemPriceMultiplierDefault = '1'
export const simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceAmountRegExpFive = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const simulateInvoiceBodyLinesItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemFeatureKeyMax = 64

export const simulateInvoiceBodyLinesItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const simulateInvoiceBodyLinesItemRateCardFeatureKeyMax = 64

export const simulateInvoiceBodyLinesItemRateCardFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const simulateInvoiceBodyLinesItemRateCardTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPricePaymentTermDefault =
  'in_advance'
export const simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMultiplierDefault = '1'
export const simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const simulateInvoiceBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const simulateInvoiceBodyLinesItemQuantityRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const simulateInvoiceBodyLinesItemPreLinePeriodQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const simulateInvoiceBodyLinesItemIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const simulateInvoiceBody = zod
  .object({
    currency: zod
      .string()
      .min(simulateInvoiceBodyCurrencyMinOne)
      .max(simulateInvoiceBodyCurrencyMaxOne)
      .regex(simulateInvoiceBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .describe(
        'Currency for all invoice line items.\n\nMulti currency invoices are not supported yet.'
      ),
    lines: zod
      .array(
        zod
          .object({
            description: zod
              .string()
              .max(simulateInvoiceBodyLinesItemDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            featureKey: zod
              .string()
              .min(1)
              .max(simulateInvoiceBodyLinesItemFeatureKeyMax)
              .regex(simulateInvoiceBodyLinesItemFeatureKeyRegExp)
              .optional()
              .describe('The feature that the usage is based on.'),
            id: zod
              .string()
              .regex(simulateInvoiceBodyLinesItemIdRegExp)
              .optional()
              .describe(
                'ID of the line. If not specified it will be auto-generated.\n\nWhen discounts are specified, this must be provided, so that the discount can reference it.'
              ),
            invoiceAt: zod
              .date()
              .describe('The time this line item should be invoiced.'),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(simulateInvoiceBodyLinesItemNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            period: zod
              .object({
                from: zod.date().describe('Period start time.'),
                to: zod.date().describe('Period end time.'),
              })
              .describe('A period with a start and end time.')
              .describe(
                'Period of the line item applies to for revenue recognition pruposes.\n\nBilling always treats periods as start being inclusive and end being exclusive.'
              ),
            preLinePeriodQuantity: zod
              .string()
              .regex(simulateInvoiceBodyLinesItemPreLinePeriodQuantityRegExpOne)
              .describe('Numeric represents an arbitrary precision number.')
              .optional()
              .describe(
                "The quantity of the item used before this line's period, if the line is billed progressively."
              ),
            price: zod
              .discriminatedUnion('type', [
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(simulateInvoiceBodyLinesItemPriceAmountRegExpOne)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the flat price.'),
                    paymentTerm: zod
                      .enum(['in_advance', 'in_arrears'])
                      .describe(
                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                      )
                      .default(
                        simulateInvoiceBodyLinesItemPricePaymentTermDefault
                      )
                      .describe(
                        'The payment term of the flat price.\nDefaults to in advance.'
                      ),
                    type: zod.enum(['flat']),
                  })
                  .describe('Flat price with payment term.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(simulateInvoiceBodyLinesItemPriceAmountRegExpThree)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the unit price.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    type: zod.enum(['unit']),
                  })
                  .describe('Unit price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    mode: zod
                      .enum(['volume', 'graduated'])
                      .describe('The mode of the tiered price.')
                      .describe(
                        'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                      ),
                    tiers: zod
                      .array(
                        zod
                          .object({
                            flatPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    simulateInvoiceBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                type: zod
                                  .enum(['flat'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Flat price.')
                              .nullable()
                              .describe(
                                'The flat price component of the tier.'
                              ),
                            unitPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    simulateInvoiceBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                type: zod
                                  .enum(['unit'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Unit price.')
                              .nullable()
                              .describe(
                                'The unit price component of the tier.'
                              ),
                            upToAmount: zod
                              .string()
                              .regex(
                                simulateInvoiceBodyLinesItemPriceTiersItemUpToAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .optional()
                              .describe(
                                'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                              ),
                          })
                          .describe(
                            'A price tier.\nAt least one price component is required in each tier.'
                          )
                      )
                      .min(1)
                      .describe(
                        'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                      ),
                    type: zod.enum(['tiered']),
                  })
                  .describe('Tiered price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    multiplier: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMultiplierRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .default(
                        simulateInvoiceBodyLinesItemPriceMultiplierDefault
                      )
                      .describe(
                        'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                      ),
                    type: zod.enum(['dynamic']),
                  })
                  .describe('Dynamic price with spend commitments.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(simulateInvoiceBodyLinesItemPriceAmountRegExpFive)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The price of one package.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMaximumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceMinimumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    quantityPerPackage: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemPriceQuantityPerPackageRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The quantity per package.'),
                    type: zod.enum(['package']),
                  })
                  .describe('Package price with spend commitments.'),
              ])
              .describe('The price of the usage based rate card.')
              .optional()
              .describe('Price of the usage-based item being sold.'),
            quantity: zod
              .string()
              .regex(simulateInvoiceBodyLinesItemQuantityRegExpOne)
              .describe('Numeric represents an arbitrary precision number.')
              .describe('The quantity of the item being sold.'),
            rateCard: zod
              .object({
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('A percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        quantity: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe('A usage discount.')
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('A discount by type.')
                  .optional()
                  .describe('The discounts that are applied to the line.'),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(simulateInvoiceBodyLinesItemRateCardFeatureKeyMax)
                  .regex(simulateInvoiceBodyLinesItemRateCardFeatureKeyRegExp)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                price: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the flat price.'),
                        paymentTerm: zod
                          .enum(['in_advance', 'in_arrears'])
                          .describe(
                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                          )
                          .default(
                            simulateInvoiceBodyLinesItemRateCardPricePaymentTermDefault
                          )
                          .describe(
                            'The payment term of the flat price.\nDefaults to in advance.'
                          ),
                        type: zod.enum(['flat']),
                      })
                      .describe('Flat price with payment term.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the unit price.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        type: zod.enum(['unit']),
                      })
                      .describe('Unit price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        mode: zod
                          .enum(['volume', 'graduated'])
                          .describe('The mode of the tiered price.')
                          .describe(
                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                          ),
                        tiers: zod
                          .array(
                            zod
                              .object({
                                flatPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        simulateInvoiceBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    type: zod
                                      .enum(['flat'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Flat price.')
                                  .nullable()
                                  .describe(
                                    'The flat price component of the tier.'
                                  ),
                                unitPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        simulateInvoiceBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the unit price.'
                                      ),
                                    type: zod
                                      .enum(['unit'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Unit price.')
                                  .nullable()
                                  .describe(
                                    'The unit price component of the tier.'
                                  ),
                                upToAmount: zod
                                  .string()
                                  .regex(
                                    simulateInvoiceBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                  ),
                              })
                              .describe(
                                'A price tier.\nAt least one price component is required in each tier.'
                              )
                          )
                          .min(1)
                          .describe(
                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                          ),
                        type: zod.enum(['tiered']),
                      })
                      .describe('Tiered price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        multiplier: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMultiplierRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .default(
                            simulateInvoiceBodyLinesItemRateCardPriceMultiplierDefault
                          )
                          .describe(
                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                          ),
                        type: zod.enum(['dynamic']),
                      })
                      .describe('Dynamic price with spend commitments.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The price of one package.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        quantityPerPackage: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The quantity per package.'),
                        type: zod.enum(['package']),
                      })
                      .describe('Package price with spend commitments.'),
                  ])
                  .describe('The price of the usage based rate card.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            simulateInvoiceBodyLinesItemRateCardTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
              })
              .describe(
                'InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line.'
              )
              .optional()
              .describe(
                'The rate card that is used for this line.\n\nThe rate card captures the intent of the price and discounts for the usage-based item.'
              ),
            taxConfig: zod
              .object({
                behavior: zod
                  .enum(['inclusive', 'exclusive'])
                  .describe(
                    'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                  )
                  .optional()
                  .describe(
                    "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                  ),
                customInvoicing: zod
                  .object({
                    code: zod
                      .string()
                      .describe(
                        'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                      ),
                  })
                  .describe('Custom invoicing tax config.')
                  .optional()
                  .describe('Custom invoicing tax config.'),
                stripe: zod
                  .object({
                    code: zod
                      .string()
                      .regex(
                        simulateInvoiceBodyLinesItemTaxConfigStripeCodeRegExp
                      )
                      .describe(
                        'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                      ),
                  })
                  .describe('The tax config for Stripe.')
                  .optional()
                  .describe('Stripe tax config.'),
              })
              .describe('Set of provider specific tax configs.')
              .optional()
              .describe(
                'Tax config specify the tax configuration for this line.'
              ),
          })
          .describe(
            'InvoiceSimulationLine represents a usage-based line item that can be input to the simulation endpoint.'
          )
      )
      .describe('Lines to be included in the generated invoice.'),
    number: zod
      .string()
      .min(1)
      .max(simulateInvoiceBodyNumberMaxOne)
      .describe(
        'InvoiceNumber is a unique identifier for the invoice, generated by the\ninvoicing app.\n\nThe uniqueness depends on a lot of factors:\n- app setting (unique per app or unique per customer)\n- multiple app scenarios (multiple apps generating invoices with the same prefix)'
      )
      .optional()
      .describe('The number of the invoice.'),
  })
  .describe('InvoiceSimulationInput is the input for simulating an invoice.')

/**
 * List invoices based on the specified filters.

The expand option can be used to include additional information (besides the invoice header and totals)
in the response. For example by adding the expand=lines option the invoice lines will be included in the response.

Gathering invoices will always show the current usage calculated on the fly.
 * @summary List invoices
 */
export const listInvoicesQueryCustomersItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listInvoicesQueryPageDefault = 1
export const listInvoicesQueryPageSizeDefault = 100
export const listInvoicesQueryPageSizeMax = 1000

export const listInvoicesQueryParams = zod.object({
  createdAfter: zod
    .date()
    .optional()
    .describe('Filter by invoice created time.\nInclusive.'),
  createdBefore: zod
    .date()
    .optional()
    .describe('Filter by invoice created time.\nInclusive.'),
  customers: zod
    .array(
      zod
        .string()
        .regex(listInvoicesQueryCustomersItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by customer ID'),
  expand: zod
    .array(
      zod
        .enum(['lines', 'preceding', 'workflow.apps'])
        .describe(
          'InvoiceExpand specifies the parts of the invoice to expand in the list output.'
        )
    )
    .optional()
    .describe('What parts of the list output to expand in listings'),
  extendedStatuses: zod
    .array(zod.string())
    .optional()
    .describe('Filter by invoice extended statuses'),
  includeDeleted: zod.boolean().optional().describe('Include deleted invoices'),
  issuedAfter: zod
    .date()
    .optional()
    .describe('Filter by invoice issued time.\nInclusive.'),
  issuedBefore: zod
    .date()
    .optional()
    .describe('Filter by invoice issued time.\nInclusive.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum([
      'customer.name',
      'issuedAt',
      'status',
      'createdAt',
      'updatedAt',
      'periodStart',
    ])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listInvoicesQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listInvoicesQueryPageSizeMax)
    .default(listInvoicesQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  periodStartAfter: zod
    .date()
    .optional()
    .describe('Filter by period start time.\nInclusive.'),
  periodStartBefore: zod
    .date()
    .optional()
    .describe('Filter by period start time.\nInclusive.'),
  statuses: zod
    .array(
      zod
        .enum([
          'gathering',
          'draft',
          'issuing',
          'issued',
          'payment_processing',
          'overdue',
          'paid',
          'uncollectible',
          'voided',
        ])
        .describe('InvoiceStatus describes the status of an invoice.')
    )
    .optional()
    .describe('Filter by the invoice status.'),
})

/**
 * Create a new invoice from the pending line items.

This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.

When creating an invoice, the pending line items will be marked as invoiced and the invoice will be created with the total amount of the pending items.

New pending line items will be created for the period between now() and the next billing cycle's begining date for any metered item.

The call can return multiple invoices if the pending line items are in different currencies.
 * @summary Invoice a customer based on the pending line items
 */
export const invoicePendingLinesActionBodyFiltersLineIdsItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const invoicePendingLinesActionBodyCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const invoicePendingLinesActionBody = zod
  .object({
    asOf: zod
      .date()
      .optional()
      .describe(
        'The time as of which the invoice is created.\n\nIf not provided, the current time is used.'
      ),
    customerId: zod
      .string()
      .regex(invoicePendingLinesActionBodyCustomerIdRegExp)
      .describe('The customer ID for which to create the invoice.'),
    filters: zod
      .object({
        lineIds: zod
          .array(
            zod
              .string()
              .regex(invoicePendingLinesActionBodyFiltersLineIdsItemRegExp)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .optional()
          .describe(
            'The pending line items to include in the invoice, if not provided:\n- all line items that have invoice_at < asOf will be included\n- [progressive billing only] all usage based line items will be included up to asOf, new\nusage-based line items will be staged for the rest of the billing cycle\n\nAll lineIDs present in the list, must exists and must be invoicable as of asOf, or the action will fail.'
          ),
      })
      .describe(
        'InvoicePendingLinesActionFiltersInput specifies which lines to include in the invoice.'
      )
      .optional()
      .describe('Filters to apply when creating the invoice.'),
    progressiveBillingOverride: zod
      .boolean()
      .optional()
      .describe(
        "Override the progressive billing setting of the customer.\n\nCan be used to disable/enable progressive billing in case the business logic\nrequires it, if not provided the billing profile's progressive billing setting will be used."
      ),
  })
  .describe(
    'BillingInvoiceActionInput is the input for creating an invoice.\n\nInvoice creation is always based on already pending line items created by the billingCreateLineByCustomer\noperation. Empty invoices are not allowed.'
  )

/**
 * Get an invoice by ID.

Gathering invoices will always show the current usage calculated on the fly.
 * @summary Get an invoice
 */
export const getInvoicePathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getInvoiceParams = zod.object({
  invoiceId: zod.string().regex(getInvoicePathInvoiceIdRegExp),
})

export const getInvoiceQueryExpandDefault = ['lines']
export const getInvoiceQueryIncludeDeletedLinesDefault = false

export const getInvoiceQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['lines', 'preceding', 'workflow.apps'])
        .describe(
          'InvoiceExpand specifies the parts of the invoice to expand in the list output.'
        )
    )
    .default(getInvoiceQueryExpandDefault),
  includeDeletedLines: zod.boolean().optional(),
})

/**
 * Delete an invoice

Only invoices that are in the draft (or earlier) status can be deleted.

Invoices that are post finalization can only be voided.
 * @summary Delete an invoice
 */
export const deleteInvoicePathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteInvoiceParams = zod.object({
  invoiceId: zod.string().regex(deleteInvoicePathInvoiceIdRegExp),
})

/**
 * Update an invoice

Only invoices in draft or earlier status can be updated.
 * @summary Update an invoice
 */
export const updateInvoicePathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateInvoiceParams = zod.object({
  invoiceId: zod.string().regex(updateInvoicePathInvoiceIdRegExp),
})

export const updateInvoiceBodyDescriptionMax = 1024
export const updateInvoiceBodySupplierTaxIdCodeMaxOne = 32
export const updateInvoiceBodySupplierAddressesItemCountryMinOne = 2

export const updateInvoiceBodySupplierAddressesItemCountryMaxOne = 2

export const updateInvoiceBodySupplierAddressesItemCountryRegExpOne =
  new RegExp('^[A-Z]{2}$')
export const updateInvoiceBodySupplierAddressesMax = 1
export const updateInvoiceBodyCustomerTaxIdCodeMaxOne = 32
export const updateInvoiceBodyCustomerAddressesItemCountryMinOne = 2

export const updateInvoiceBodyCustomerAddressesItemCountryMaxOne = 2

export const updateInvoiceBodyCustomerAddressesItemCountryRegExpOne =
  new RegExp('^[A-Z]{2}$')
export const updateInvoiceBodyCustomerAddressesMax = 1
export const updateInvoiceBodyLinesItemNameMax = 256
export const updateInvoiceBodyLinesItemDescriptionMax = 1024
export const updateInvoiceBodyLinesItemTaxConfigStripeCodeRegExp = new RegExp(
  '^txcd_\\d{8}$'
)
export const updateInvoiceBodyLinesItemPriceAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPricePaymentTermDefault = 'in_advance'
export const updateInvoiceBodyLinesItemPriceAmountRegExpThree = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPriceMinimumAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPriceMaximumAmountRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMultiplierRegExpOne = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPriceMultiplierDefault = '1'
export const updateInvoiceBodyLinesItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceAmountRegExpFive = new RegExp(
  '^\\-?[0-9]+(\\.[0-9]+)?$'
)
export const updateInvoiceBodyLinesItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemFeatureKeyMax = 64

export const updateInvoiceBodyLinesItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateInvoiceBodyLinesItemRateCardFeatureKeyMax = 64

export const updateInvoiceBodyLinesItemRateCardFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updateInvoiceBodyLinesItemRateCardTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const updateInvoiceBodyLinesItemRateCardPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPricePaymentTermDefault =
  'in_advance'
export const updateInvoiceBodyLinesItemRateCardPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMultiplierDefault = '1'
export const updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const updateInvoiceBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updateInvoiceBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp =
  new RegExp('^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$')
export const updateInvoiceBodyLinesItemIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateInvoiceBodyWorkflowWorkflowInvoicingAutoAdvanceDefault = true
export const updateInvoiceBodyWorkflowWorkflowInvoicingDraftPeriodDefault =
  'P0D'
export const updateInvoiceBodyWorkflowWorkflowInvoicingDueAfterDefault = 'P30D'
export const updateInvoiceBodyWorkflowWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const updateInvoiceBodyWorkflowWorkflowPaymentCollectionMethodDefault =
  'charge_automatically'

export const updateInvoiceBody = zod
  .object({
    customer: zod
      .object({
        addresses: zod
          .array(
            zod
              .object({
                city: zod.string().optional().describe('City.'),
                country: zod
                  .string()
                  .min(updateInvoiceBodyCustomerAddressesItemCountryMinOne)
                  .max(updateInvoiceBodyCustomerAddressesItemCountryMaxOne)
                  .regex(updateInvoiceBodyCustomerAddressesItemCountryRegExpOne)
                  .describe(
                    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
                  )
                  .optional()
                  .describe(
                    'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
                  ),
                line1: zod
                  .string()
                  .optional()
                  .describe('First line of the address.'),
                line2: zod
                  .string()
                  .optional()
                  .describe('Second line of the address.'),
                phoneNumber: zod.string().optional().describe('Phone number.'),
                postalCode: zod.string().optional().describe('Postal code.'),
                state: zod.string().optional().describe('State or province.'),
              })
              .describe('Address')
          )
          .max(updateInvoiceBodyCustomerAddressesMax)
          .optional()
          .describe(
            'Regular post addresses for where information should be sent if needed.'
          ),
        name: zod
          .string()
          .optional()
          .describe('Legal name or representation of the organization.'),
        taxId: zod
          .object({
            code: zod
              .string()
              .min(1)
              .max(updateInvoiceBodyCustomerTaxIdCodeMaxOne)
              .describe(
                'TaxIdentificationCode is a normalized tax code shown on the original identity document.'
              )
              .optional()
              .describe(
                'Normalized tax code shown on the original identity document.'
              ),
          })
          .describe(
            'Identity stores the details required to identify an entity for tax purposes in a specific country.'
          )
          .optional()
          .describe(
            "The entity's legal ID code used for tax purposes. They may have\nother numbers, but we're only interested in those valid for tax purposes."
          ),
      })
      .describe('Resource update operation model.')
      .describe('The customer the invoice is sent to.'),
    description: zod
      .string()
      .max(updateInvoiceBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    lines: zod
      .array(
        zod
          .object({
            description: zod
              .string()
              .max(updateInvoiceBodyLinesItemDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            featureKey: zod
              .string()
              .min(1)
              .max(updateInvoiceBodyLinesItemFeatureKeyMax)
              .regex(updateInvoiceBodyLinesItemFeatureKeyRegExp)
              .optional()
              .describe('The feature that the usage is based on.'),
            id: zod
              .string()
              .regex(updateInvoiceBodyLinesItemIdRegExp)
              .optional()
              .describe('The ID of the line.'),
            invoiceAt: zod
              .date()
              .describe('The time this line item should be invoiced.'),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(updateInvoiceBodyLinesItemNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            period: zod
              .object({
                from: zod.date().describe('Period start time.'),
                to: zod.date().describe('Period end time.'),
              })
              .describe('A period with a start and end time.')
              .describe(
                'Period of the line item applies to for revenue recognition pruposes.\n\nBilling always treats periods as start being inclusive and end being exclusive.'
              ),
            price: zod
              .discriminatedUnion('type', [
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(updateInvoiceBodyLinesItemPriceAmountRegExpOne)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the flat price.'),
                    paymentTerm: zod
                      .enum(['in_advance', 'in_arrears'])
                      .describe(
                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                      )
                      .default(
                        updateInvoiceBodyLinesItemPricePaymentTermDefault
                      )
                      .describe(
                        'The payment term of the flat price.\nDefaults to in advance.'
                      ),
                    type: zod.enum(['flat']),
                  })
                  .describe('Flat price with payment term.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(updateInvoiceBodyLinesItemPriceAmountRegExpThree)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The amount of the unit price.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMaximumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMinimumAmountRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    type: zod.enum(['unit']),
                  })
                  .describe('Unit price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMaximumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMinimumAmountRegExpThree
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    mode: zod
                      .enum(['volume', 'graduated'])
                      .describe('The mode of the tiered price.')
                      .describe(
                        'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                      ),
                    tiers: zod
                      .array(
                        zod
                          .object({
                            flatPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    updateInvoiceBodyLinesItemPriceTiersItemFlatPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                type: zod
                                  .enum(['flat'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Flat price.')
                              .nullable()
                              .describe(
                                'The flat price component of the tier.'
                              ),
                            unitPrice: zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    updateInvoiceBodyLinesItemPriceTiersItemUnitPriceAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                type: zod
                                  .enum(['unit'])
                                  .describe('The type of the price.'),
                              })
                              .describe('Unit price.')
                              .nullable()
                              .describe(
                                'The unit price component of the tier.'
                              ),
                            upToAmount: zod
                              .string()
                              .regex(
                                updateInvoiceBodyLinesItemPriceTiersItemUpToAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .optional()
                              .describe(
                                'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                              ),
                          })
                          .describe(
                            'A price tier.\nAt least one price component is required in each tier.'
                          )
                      )
                      .min(1)
                      .describe(
                        'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                      ),
                    type: zod.enum(['tiered']),
                  })
                  .describe('Tiered price with spend commitments.'),
                zod
                  .object({
                    maximumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMaximumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMinimumAmountRegExpFive
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    multiplier: zod
                      .string()
                      .regex(updateInvoiceBodyLinesItemPriceMultiplierRegExpOne)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .default(updateInvoiceBodyLinesItemPriceMultiplierDefault)
                      .describe(
                        'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                      ),
                    type: zod.enum(['dynamic']),
                  })
                  .describe('Dynamic price with spend commitments.'),
                zod
                  .object({
                    amount: zod
                      .string()
                      .regex(updateInvoiceBodyLinesItemPriceAmountRegExpFive)
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The price of one package.'),
                    maximumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMaximumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is limited to spend at most the amount.'
                      ),
                    minimumAmount: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceMinimumAmountRegExpSeven
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .optional()
                      .describe(
                        'The customer is committed to spend at least the amount.'
                      ),
                    quantityPerPackage: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemPriceQuantityPerPackageRegExpOne
                      )
                      .describe(
                        'Numeric represents an arbitrary precision number.'
                      )
                      .describe('The quantity per package.'),
                    type: zod.enum(['package']),
                  })
                  .describe('Package price with spend commitments.'),
              ])
              .describe('The price of the usage based rate card.')
              .optional()
              .describe('Price of the usage-based item being sold.'),
            rateCard: zod
              .object({
                discounts: zod
                  .object({
                    percentage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardDiscountsPercentageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        percentage: zod
                          .number()
                          .describe(
                            'Numeric representation of a percentage\n\n50% is represented as 50'
                          )
                          .describe('The percentage of the discount.'),
                      })
                      .describe('A percentage discount.')
                      .optional()
                      .describe('The percentage discount.'),
                    usage: zod
                      .object({
                        correlationId: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardDiscountsUsageCorrelationIdRegExp
                          )
                          .optional()
                          .describe(
                            'Correlation ID for the discount.\n\nThis is used to link discounts across different invoices (progressive billing use case).\n\nIf not provided, the invoicing engine will auto-generate one. When editing an invoice line,\nplease make sure to keep the same correlation ID of the discount or in progressive billing\nsetups the discount amounts might be incorrect.'
                          ),
                        quantity: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardDiscountsUsageQuantityRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe(
                            'The quantity of the usage discount.\n\nMust be positive.'
                          ),
                      })
                      .describe('A usage discount.')
                      .optional()
                      .describe('The usage discount.'),
                  })
                  .describe('A discount by type.')
                  .optional()
                  .describe('The discounts that are applied to the line.'),
                featureKey: zod
                  .string()
                  .min(1)
                  .max(updateInvoiceBodyLinesItemRateCardFeatureKeyMax)
                  .regex(updateInvoiceBodyLinesItemRateCardFeatureKeyRegExp)
                  .optional()
                  .describe('The feature the customer is entitled to use.'),
                price: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the flat price.'),
                        paymentTerm: zod
                          .enum(['in_advance', 'in_arrears'])
                          .describe(
                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                          )
                          .default(
                            updateInvoiceBodyLinesItemRateCardPricePaymentTermDefault
                          )
                          .describe(
                            'The payment term of the flat price.\nDefaults to in advance.'
                          ),
                        type: zod.enum(['flat']),
                      })
                      .describe('Flat price with payment term.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The amount of the unit price.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        type: zod.enum(['unit']),
                      })
                      .describe('Unit price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpThree
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        mode: zod
                          .enum(['volume', 'graduated'])
                          .describe('The mode of the tiered price.')
                          .describe(
                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                          ),
                        tiers: zod
                          .array(
                            zod
                              .object({
                                flatPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        updateInvoiceBodyLinesItemRateCardPriceTiersItemFlatPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    type: zod
                                      .enum(['flat'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Flat price.')
                                  .nullable()
                                  .describe(
                                    'The flat price component of the tier.'
                                  ),
                                unitPrice: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        updateInvoiceBodyLinesItemRateCardPriceTiersItemUnitPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the unit price.'
                                      ),
                                    type: zod
                                      .enum(['unit'])
                                      .describe('The type of the price.'),
                                  })
                                  .describe('Unit price.')
                                  .nullable()
                                  .describe(
                                    'The unit price component of the tier.'
                                  ),
                                upToAmount: zod
                                  .string()
                                  .regex(
                                    updateInvoiceBodyLinesItemRateCardPriceTiersItemUpToAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                  ),
                              })
                              .describe(
                                'A price tier.\nAt least one price component is required in each tier.'
                              )
                          )
                          .min(1)
                          .describe(
                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                          ),
                        type: zod.enum(['tiered']),
                      })
                      .describe('Tiered price with spend commitments.'),
                    zod
                      .object({
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        multiplier: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMultiplierRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .default(
                            updateInvoiceBodyLinesItemRateCardPriceMultiplierDefault
                          )
                          .describe(
                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                          ),
                        type: zod.enum(['dynamic']),
                      })
                      .describe('Dynamic price with spend commitments.'),
                    zod
                      .object({
                        amount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceAmountRegExpFive
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The price of one package.'),
                        maximumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMaximumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is limited to spend at most the amount.'
                          ),
                        minimumAmount: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceMinimumAmountRegExpSeven
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .optional()
                          .describe(
                            'The customer is committed to spend at least the amount.'
                          ),
                        quantityPerPackage: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardPriceQuantityPerPackageRegExpOne
                          )
                          .describe(
                            'Numeric represents an arbitrary precision number.'
                          )
                          .describe('The quantity per package.'),
                        type: zod.enum(['package']),
                      })
                      .describe('Package price with spend commitments.'),
                  ])
                  .describe('The price of the usage based rate card.')
                  .nullable()
                  .describe(
                    'The price of the rate card.\nWhen null, the feature or service is free.'
                  ),
                taxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            updateInvoiceBodyLinesItemRateCardTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                  ),
              })
              .describe(
                'InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based line.'
              )
              .optional()
              .describe(
                'The rate card that is used for this line.\n\nThe rate card captures the intent of the price and discounts for the usage-based item.'
              ),
            taxConfig: zod
              .object({
                behavior: zod
                  .enum(['inclusive', 'exclusive'])
                  .describe(
                    'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                  )
                  .optional()
                  .describe(
                    "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                  ),
                customInvoicing: zod
                  .object({
                    code: zod
                      .string()
                      .describe(
                        'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                      ),
                  })
                  .describe('Custom invoicing tax config.')
                  .optional()
                  .describe('Custom invoicing tax config.'),
                stripe: zod
                  .object({
                    code: zod
                      .string()
                      .regex(
                        updateInvoiceBodyLinesItemTaxConfigStripeCodeRegExp
                      )
                      .describe(
                        'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                      ),
                  })
                  .describe('The tax config for Stripe.')
                  .optional()
                  .describe('Stripe tax config.'),
              })
              .describe('Set of provider specific tax configs.')
              .optional()
              .describe(
                'Tax config specify the tax configuration for this line.'
              ),
          })
          .describe(
            'InvoiceLineReplaceUpdate represents the update model for an UBP invoice line.\n\nThis type makes ID optional to allow for creating new lines as part of the update.'
          )
      )
      .describe('The lines included in the invoice.'),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    supplier: zod
      .object({
        addresses: zod
          .array(
            zod
              .object({
                city: zod.string().optional().describe('City.'),
                country: zod
                  .string()
                  .min(updateInvoiceBodySupplierAddressesItemCountryMinOne)
                  .max(updateInvoiceBodySupplierAddressesItemCountryMaxOne)
                  .regex(updateInvoiceBodySupplierAddressesItemCountryRegExpOne)
                  .describe(
                    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
                  )
                  .optional()
                  .describe(
                    'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
                  ),
                line1: zod
                  .string()
                  .optional()
                  .describe('First line of the address.'),
                line2: zod
                  .string()
                  .optional()
                  .describe('Second line of the address.'),
                phoneNumber: zod.string().optional().describe('Phone number.'),
                postalCode: zod.string().optional().describe('Postal code.'),
                state: zod.string().optional().describe('State or province.'),
              })
              .describe('Address')
          )
          .max(updateInvoiceBodySupplierAddressesMax)
          .optional()
          .describe(
            'Regular post addresses for where information should be sent if needed.'
          ),
        name: zod
          .string()
          .optional()
          .describe('Legal name or representation of the organization.'),
        taxId: zod
          .object({
            code: zod
              .string()
              .min(1)
              .max(updateInvoiceBodySupplierTaxIdCodeMaxOne)
              .describe(
                'TaxIdentificationCode is a normalized tax code shown on the original identity document.'
              )
              .optional()
              .describe(
                'Normalized tax code shown on the original identity document.'
              ),
          })
          .describe(
            'Identity stores the details required to identify an entity for tax purposes in a specific country.'
          )
          .optional()
          .describe(
            "The entity's legal ID code used for tax purposes. They may have\nother numbers, but we're only interested in those valid for tax purposes."
          ),
      })
      .describe('Resource update operation model.')
      .describe('The supplier of the lines included in the invoice.'),
    workflow: zod
      .object({
        workflow: zod
          .object({
            invoicing: zod
              .object({
                autoAdvance: zod
                  .boolean()
                  .default(
                    updateInvoiceBodyWorkflowWorkflowInvoicingAutoAdvanceDefault
                  )
                  .describe(
                    'Whether to automatically issue the invoice after the draftPeriod has passed.'
                  ),
                defaultTaxConfig: zod
                  .object({
                    behavior: zod
                      .enum(['inclusive', 'exclusive'])
                      .describe(
                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                      )
                      .optional()
                      .describe(
                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                      ),
                    customInvoicing: zod
                      .object({
                        code: zod
                          .string()
                          .describe(
                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                          ),
                      })
                      .describe('Custom invoicing tax config.')
                      .optional()
                      .describe('Custom invoicing tax config.'),
                    stripe: zod
                      .object({
                        code: zod
                          .string()
                          .regex(
                            updateInvoiceBodyWorkflowWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp
                          )
                          .describe(
                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                          ),
                      })
                      .describe('The tax config for Stripe.')
                      .optional()
                      .describe('Stripe tax config.'),
                  })
                  .describe('Set of provider specific tax configs.')
                  .optional()
                  .describe(
                    'Default tax configuration to apply to the invoices.'
                  ),
                draftPeriod: zod
                  .string()
                  .default(
                    updateInvoiceBodyWorkflowWorkflowInvoicingDraftPeriodDefault
                  )
                  .describe(
                    'The period for the invoice to be kept in draft status for manual reviews.'
                  ),
                dueAfter: zod
                  .string()
                  .default(
                    updateInvoiceBodyWorkflowWorkflowInvoicingDueAfterDefault
                  )
                  .describe(
                    "The period after which the invoice is due.\nWith some payment solutions it's only applicable for manual collection method."
                  ),
              })
              .describe(
                'InvoiceWorkflowInvoicingSettingsReplaceUpdate represents the update model for the invoicing settings of an invoice workflow.'
              )
              .describe('The invoicing settings for this workflow'),
            payment: zod
              .object({
                collectionMethod: zod
                  .enum(['charge_automatically', 'send_invoice'])
                  .describe(
                    'CollectionMethod specifies how the invoice should be collected (automatic vs manual)'
                  )
                  .default(
                    updateInvoiceBodyWorkflowWorkflowPaymentCollectionMethodDefault
                  )
                  .describe('The payment method for the invoice.'),
              })
              .describe(
                'BillingWorkflowPaymentSettings represents the payment settings for a billing workflow'
              )
              .describe('The payment settings for this workflow'),
          })
          .describe(
            "Mutable workflow settings for an invoice.\n\nOther fields on the invoice's workflow are not mutable, they serve as a history of the invoice's workflow\nat creation time."
          )
          .describe('The workflow used for this invoice.'),
      })
      .describe(
        'InvoiceWorkflowReplaceUpdate represents the update model for an invoice workflow.\n\nFields that are immutable a re removed from the model. This is based on InvoiceWorkflowSettings.'
      )
      .describe('The workflow settings for the invoice.'),
  })
  .describe('InvoiceReplaceUpdate represents the update model for an invoice.')

/**
 * Advance the invoice's state to the next status.

The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.

The action can be called when the invoice's statusDetails' actions field contain the "advance" action.
 * @summary Advance the invoice's state to the next status
 */
export const advanceInvoiceActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const advanceInvoiceActionParams = zod.object({
  invoiceId: zod.string().regex(advanceInvoiceActionPathInvoiceIdRegExp),
})

/**
 * Approve an invoice and start executing the payment workflow.

This call instantly sends the invoice to the customer using the configured billing profile app.

This call is valid in two invoice statuses:
- `draft`: the invoice will be sent to the customer, the invluce state becomes issued
- `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued
 * @summary Send the invoice to the customer
 */
export const approveInvoiceActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const approveInvoiceActionParams = zod.object({
  invoiceId: zod.string().regex(approveInvoiceActionPathInvoiceIdRegExp),
})

/**
 * Retry advancing the invoice after a failed attempt.

The action can be called when the invoice's statusDetails' actions field contain the "retry" action.
 * @summary Retry advancing the invoice after a failed attempt.
 */
export const retryInvoiceActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const retryInvoiceActionParams = zod.object({
  invoiceId: zod.string().regex(retryInvoiceActionPathInvoiceIdRegExp),
})

/**
 * Snapshot quantities for usage based line items.

This call will snapshot the quantities for all usage based line items in the invoice.

This call is only valid in `draft.waiting_for_collection` status, where the collection period
can be skipped using this action.
 * @summary Snapshot quantities for usage based line items
 */
export const snapshotQuantitiesInvoiceActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const snapshotQuantitiesInvoiceActionParams = zod.object({
  invoiceId: zod
    .string()
    .regex(snapshotQuantitiesInvoiceActionPathInvoiceIdRegExp),
})

/**
 * Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)

Note: charges might apply, depending on the tax provider.
 * @summary Recalculate an invoice's tax amounts
 */
export const recalculateInvoiceTaxActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const recalculateInvoiceTaxActionParams = zod.object({
  invoiceId: zod.string().regex(recalculateInvoiceTaxActionPathInvoiceIdRegExp),
})

/**
 * Void an invoice

Only invoices that have been alread issued can be voided.

Voiding an invoice will mark it as voided, the user can specify how to handle the voided line items.
 * @summary Void an invoice
 */
export const voidInvoiceActionPathInvoiceIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const voidInvoiceActionParams = zod.object({
  invoiceId: zod.string().regex(voidInvoiceActionPathInvoiceIdRegExp),
})

export const voidInvoiceActionBodyOverridesItemLineIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const voidInvoiceActionBody = zod
  .object({
    action: zod
      .object({
        action: zod
          .discriminatedUnion('type', [
            zod
              .object({
                type: zod.enum(['discard']),
              })
              .describe(
                'VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.'
              ),
            zod
              .object({
                nextInvoiceAt: zod
                  .date()
                  .optional()
                  .describe(
                    'The time at which the line item should be invoiced again.\n\nIf not provided, the line item will be re-invoiced now.'
                  ),
                type: zod.enum(['pending']),
              })
              .describe(
                'VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.'
              ),
          ])
          .describe(
            'VoidInvoiceLineAction describes how to handle a specific line item in the invoice when voiding.'
          )
          .describe('The action to take on the line items.'),
        percentage: zod
          .number()
          .describe(
            'Numeric representation of a percentage\n\n50% is represented as 50'
          )
          .describe(
            'How much of the total line items to be voided? (e.g. 100% means all charges are voided)'
          ),
      })
      .describe(
        'InvoiceVoidAction describes how to handle the voided line items.'
      )
      .describe('The action to take on the voided line items.'),
    overrides: zod
      .array(
        zod
          .object({
            action: zod
              .object({
                action: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        type: zod.enum(['discard']),
                      })
                      .describe(
                        'VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.'
                      ),
                    zod
                      .object({
                        nextInvoiceAt: zod
                          .date()
                          .optional()
                          .describe(
                            'The time at which the line item should be invoiced again.\n\nIf not provided, the line item will be re-invoiced now.'
                          ),
                        type: zod.enum(['pending']),
                      })
                      .describe(
                        'VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.'
                      ),
                  ])
                  .describe(
                    'VoidInvoiceLineAction describes how to handle a specific line item in the invoice when voiding.'
                  )
                  .describe('The action to take on the line items.'),
                percentage: zod
                  .number()
                  .describe(
                    'Numeric representation of a percentage\n\n50% is represented as 50'
                  )
                  .describe(
                    'How much of the total line items to be voided? (e.g. 100% means all charges are voided)'
                  ),
              })
              .describe(
                'InvoiceVoidAction describes how to handle the voided line items.'
              )
              .describe('The action to take on the line item.'),
            lineId: zod
              .string()
              .regex(voidInvoiceActionBodyOverridesItemLineIdRegExp)
              .describe('The line item ID to override.'),
          })
          .describe(
            'VoidInvoiceLineOverride describes how to handle a specific line item in the invoice when voiding.'
          )
      )
      .nullish()
      .describe(
        'Per line item overrides for the action.\n\nIf not specified, the `action` will be applied to all line items.'
      ),
    reason: zod.string().describe('The reason for voiding the invoice.'),
  })
  .describe('Request to void an invoice')

/**
 * List all billing profiles matching the specified filters.

The expand option can be used to include additional information (besides the billing profile)
in the response. For example by adding the expand=apps option the apps used by the billing profile
will be included in the response.
 * @summary List billing profiles
 */
export const listBillingProfilesQueryIncludeArchivedDefault = false
export const listBillingProfilesQueryPageDefault = 1
export const listBillingProfilesQueryPageSizeDefault = 100
export const listBillingProfilesQueryPageSizeMax = 1000

export const listBillingProfilesQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['apps'])
        .describe('BillingProfileExpand details what profile fields to expand')
    )
    .optional(),
  includeArchived: zod.boolean().optional(),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['createdAt', 'updatedAt', 'default', 'name'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listBillingProfilesQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listBillingProfilesQueryPageSizeMax)
    .default(listBillingProfilesQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Create a new billing profile

Billing profiles are representations of a customer's billing information. Customer overrides
can be applied to a billing profile to customize the billing behavior for a specific customer.
 * @summary Create a new billing profile
 */
export const createBillingProfileBodyNameMax = 256
export const createBillingProfileBodyDescriptionMax = 1024
export const createBillingProfileBodySupplierTaxIdCodeMaxOne = 32
export const createBillingProfileBodySupplierAddressesItemCountryMinOne = 2

export const createBillingProfileBodySupplierAddressesItemCountryMaxOne = 2

export const createBillingProfileBodySupplierAddressesItemCountryRegExpOne =
  new RegExp('^[A-Z]{2}$')
export const createBillingProfileBodySupplierAddressesMax = 1
export const createBillingProfileBodyWorkflowCollectionAlignmentDefault = {
  type: 'subscription',
}
export const createBillingProfileBodyWorkflowCollectionIntervalDefault = 'PT1H'
export const createBillingProfileBodyWorkflowInvoicingAutoAdvanceDefault = true
export const createBillingProfileBodyWorkflowInvoicingDraftPeriodDefault = 'P0D'
export const createBillingProfileBodyWorkflowInvoicingDueAfterDefault = 'P30D'
export const createBillingProfileBodyWorkflowInvoicingProgressiveBillingDefault =
  false
export const createBillingProfileBodyWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const createBillingProfileBodyWorkflowPaymentCollectionMethodDefault =
  'charge_automatically'
export const createBillingProfileBodyWorkflowTaxEnabledDefault = true
export const createBillingProfileBodyWorkflowTaxEnforcedDefault = false
export const createBillingProfileBodyAppsTaxRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createBillingProfileBodyAppsInvoicingRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createBillingProfileBodyAppsPaymentRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createBillingProfileBody = zod
  .object({
    apps: zod
      .object({
        invoicing: zod
          .string()
          .regex(createBillingProfileBodyAppsInvoicingRegExpOne)
          .describe(
            'ULID (Universally Unique Lexicographically Sortable Identifier).'
          )
          .or(zod.string())
          .describe('The invoicing app used for this workflow'),
        payment: zod
          .string()
          .regex(createBillingProfileBodyAppsPaymentRegExpOne)
          .describe(
            'ULID (Universally Unique Lexicographically Sortable Identifier).'
          )
          .or(zod.string())
          .describe('The payment app used for this workflow'),
        tax: zod
          .string()
          .regex(createBillingProfileBodyAppsTaxRegExpOne)
          .describe(
            'ULID (Universally Unique Lexicographically Sortable Identifier).'
          )
          .or(zod.string())
          .describe('The tax app used for this workflow'),
      })
      .describe(
        "BillingProfileAppsCreate represents the input for creating a billing profile's apps"
      )
      .describe('The apps used by this billing profile.'),
    default: zod.boolean().describe('Is this the default profile?'),
    description: zod
      .string()
      .max(createBillingProfileBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createBillingProfileBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    supplier: zod
      .object({
        addresses: zod
          .array(
            zod
              .object({
                city: zod.string().optional().describe('City.'),
                country: zod
                  .string()
                  .min(
                    createBillingProfileBodySupplierAddressesItemCountryMinOne
                  )
                  .max(
                    createBillingProfileBodySupplierAddressesItemCountryMaxOne
                  )
                  .regex(
                    createBillingProfileBodySupplierAddressesItemCountryRegExpOne
                  )
                  .describe(
                    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
                  )
                  .optional()
                  .describe(
                    'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
                  ),
                line1: zod
                  .string()
                  .optional()
                  .describe('First line of the address.'),
                line2: zod
                  .string()
                  .optional()
                  .describe('Second line of the address.'),
                phoneNumber: zod.string().optional().describe('Phone number.'),
                postalCode: zod.string().optional().describe('Postal code.'),
                state: zod.string().optional().describe('State or province.'),
              })
              .describe('Address')
          )
          .max(createBillingProfileBodySupplierAddressesMax)
          .optional()
          .describe(
            'Regular post addresses for where information should be sent if needed.'
          ),
        id: zod
          .string()
          .optional()
          .describe('Unique identifier for the party (if available)'),
        name: zod
          .string()
          .optional()
          .describe('Legal name or representation of the organization.'),
        taxId: zod
          .object({
            code: zod
              .string()
              .min(1)
              .max(createBillingProfileBodySupplierTaxIdCodeMaxOne)
              .describe(
                'TaxIdentificationCode is a normalized tax code shown on the original identity document.'
              )
              .optional()
              .describe(
                'Normalized tax code shown on the original identity document.'
              ),
          })
          .describe(
            'Identity stores the details required to identify an entity for tax purposes in a specific country.'
          )
          .optional()
          .describe(
            "The entity's legal ID code used for tax purposes. They may have\nother numbers, but we're only interested in those valid for tax purposes."
          ),
      })
      .describe('Party represents a person or business entity.')
      .describe(
        'The name and contact information for the supplier this billing profile represents'
      ),
    workflow: zod
      .object({
        collection: zod
          .object({
            alignment: zod
              .object({
                type: zod
                  .enum(['subscription'])
                  .describe('The type of alignment.'),
              })
              .describe(
                'BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items\ninto an invoice.'
              )
              .describe(
                'The alignment for collecting the pending line items into an invoice.\n\nDefaults to subscription, which means that we are to create a new invoice every time the\na subscription period starts (for in advance items) or ends (for in arrears items).'
              )
              .default(
                createBillingProfileBodyWorkflowCollectionAlignmentDefault
              )
              .describe(
                'The alignment for collecting the pending line items into an invoice.'
              ),
            interval: zod
              .string()
              .default(
                createBillingProfileBodyWorkflowCollectionIntervalDefault
              )
              .describe(
                'This grace period can be used to delay the collection of the pending line items specified in\nalignment.\n\nThis is useful, in case of multiple subscriptions having slightly different billing periods.'
              ),
          })
          .describe(
            'Workflow collection specifies how to collect the pending line items for an invoice'
          )
          .optional()
          .describe('The collection settings for this workflow'),
        invoicing: zod
          .object({
            autoAdvance: zod
              .boolean()
              .default(
                createBillingProfileBodyWorkflowInvoicingAutoAdvanceDefault
              )
              .describe(
                'Whether to automatically issue the invoice after the draftPeriod has passed.'
              ),
            defaultTaxConfig: zod
              .object({
                behavior: zod
                  .enum(['inclusive', 'exclusive'])
                  .describe(
                    'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                  )
                  .optional()
                  .describe(
                    "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                  ),
                customInvoicing: zod
                  .object({
                    code: zod
                      .string()
                      .describe(
                        'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                      ),
                  })
                  .describe('Custom invoicing tax config.')
                  .optional()
                  .describe('Custom invoicing tax config.'),
                stripe: zod
                  .object({
                    code: zod
                      .string()
                      .regex(
                        createBillingProfileBodyWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp
                      )
                      .describe(
                        'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                      ),
                  })
                  .describe('The tax config for Stripe.')
                  .optional()
                  .describe('Stripe tax config.'),
              })
              .describe('Set of provider specific tax configs.')
              .optional()
              .describe('Default tax configuration to apply to the invoices.'),
            draftPeriod: zod
              .string()
              .default(
                createBillingProfileBodyWorkflowInvoicingDraftPeriodDefault
              )
              .describe(
                'The period for the invoice to be kept in draft status for manual reviews.'
              ),
            dueAfter: zod
              .string()
              .default(createBillingProfileBodyWorkflowInvoicingDueAfterDefault)
              .describe(
                "The period after which the invoice is due.\nWith some payment solutions it's only applicable for manual collection method."
              ),
            progressiveBilling: zod
              .boolean()
              .optional()
              .describe(
                'Should progressive billing be allowed for this workflow?'
              ),
          })
          .describe(
            'BillingWorkflowInvoicingSettings represents the invoice settings for a billing workflow'
          )
          .optional()
          .describe('The invoicing settings for this workflow'),
        payment: zod
          .object({
            collectionMethod: zod
              .enum(['charge_automatically', 'send_invoice'])
              .describe(
                'CollectionMethod specifies how the invoice should be collected (automatic vs manual)'
              )
              .default(
                createBillingProfileBodyWorkflowPaymentCollectionMethodDefault
              )
              .describe('The payment method for the invoice.'),
          })
          .describe(
            'BillingWorkflowPaymentSettings represents the payment settings for a billing workflow'
          )
          .optional()
          .describe('The payment settings for this workflow'),
        tax: zod
          .object({
            enabled: zod
              .boolean()
              .default(createBillingProfileBodyWorkflowTaxEnabledDefault)
              .describe(
                'Enable automatic tax calculation when tax is supported by the app.\nFor example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.'
              ),
            enforced: zod
              .boolean()
              .optional()
              .describe(
                'Enforce tax calculation when tax is supported by the app.\nWhen enabled, OpenMeter will not allow to create an invoice without tax calculation.\nEnforcement is different per apps, for example, Stripe app requires customer\nto have a tax location when starting a paid subscription.'
              ),
          })
          .describe(
            'BillingWorkflowTaxSettings represents the tax settings for a billing workflow'
          )
          .optional()
          .describe('The tax settings for this workflow'),
      })
      .describe('Resource create operation model.')
      .describe('The billing workflow settings for this profile.'),
  })
  .describe(
    'BillingProfileCreate represents the input for creating a billing profile'
  )

/**
 * Delete a billing profile by id.

Only such billing profiles can be deleted that are:
- not the default one
- not pinned to any customer using customer overrides
- only have finalized invoices
 * @summary Delete a billing profile
 */
export const deleteBillingProfilePathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteBillingProfileParams = zod.object({
  id: zod.string().regex(deleteBillingProfilePathIdRegExp),
})

/**
 * Get a billing profile by id.

The expand option can be used to include additional information (besides the billing profile)
in the response. For example by adding the expand=apps option the apps used by the billing profile
will be included in the response.
 * @summary Get a billing profile
 */
export const getBillingProfilePathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getBillingProfileParams = zod.object({
  id: zod.string().regex(getBillingProfilePathIdRegExp),
})

export const getBillingProfileQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['apps'])
        .describe('BillingProfileExpand details what profile fields to expand')
    )
    .optional(),
})

/**
 * Update a billing profile by id.

The apps field cannot be updated directly, if an app change is desired a new
profile should be created.
 * @summary Update a billing profile
 */
export const updateBillingProfilePathIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateBillingProfileParams = zod.object({
  id: zod.string().regex(updateBillingProfilePathIdRegExp),
})

export const updateBillingProfileBodyNameMax = 256
export const updateBillingProfileBodyDescriptionMax = 1024
export const updateBillingProfileBodySupplierTaxIdCodeMaxOne = 32
export const updateBillingProfileBodySupplierAddressesItemCountryMinOne = 2

export const updateBillingProfileBodySupplierAddressesItemCountryMaxOne = 2

export const updateBillingProfileBodySupplierAddressesItemCountryRegExpOne =
  new RegExp('^[A-Z]{2}$')
export const updateBillingProfileBodySupplierAddressesMax = 1
export const updateBillingProfileBodyWorkflowCollectionAlignmentDefault = {
  type: 'subscription',
}
export const updateBillingProfileBodyWorkflowCollectionIntervalDefault = 'PT1H'
export const updateBillingProfileBodyWorkflowInvoicingAutoAdvanceDefault = true
export const updateBillingProfileBodyWorkflowInvoicingDraftPeriodDefault = 'P0D'
export const updateBillingProfileBodyWorkflowInvoicingDueAfterDefault = 'P30D'
export const updateBillingProfileBodyWorkflowInvoicingProgressiveBillingDefault =
  false
export const updateBillingProfileBodyWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const updateBillingProfileBodyWorkflowPaymentCollectionMethodDefault =
  'charge_automatically'
export const updateBillingProfileBodyWorkflowTaxEnabledDefault = true
export const updateBillingProfileBodyWorkflowTaxEnforcedDefault = false

export const updateBillingProfileBody = zod
  .object({
    default: zod.boolean().describe('Is this the default profile?'),
    description: zod
      .string()
      .max(updateBillingProfileBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updateBillingProfileBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    supplier: zod
      .object({
        addresses: zod
          .array(
            zod
              .object({
                city: zod.string().optional().describe('City.'),
                country: zod
                  .string()
                  .min(
                    updateBillingProfileBodySupplierAddressesItemCountryMinOne
                  )
                  .max(
                    updateBillingProfileBodySupplierAddressesItemCountryMaxOne
                  )
                  .regex(
                    updateBillingProfileBodySupplierAddressesItemCountryRegExpOne
                  )
                  .describe(
                    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
                  )
                  .optional()
                  .describe(
                    'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
                  ),
                line1: zod
                  .string()
                  .optional()
                  .describe('First line of the address.'),
                line2: zod
                  .string()
                  .optional()
                  .describe('Second line of the address.'),
                phoneNumber: zod.string().optional().describe('Phone number.'),
                postalCode: zod.string().optional().describe('Postal code.'),
                state: zod.string().optional().describe('State or province.'),
              })
              .describe('Address')
          )
          .max(updateBillingProfileBodySupplierAddressesMax)
          .optional()
          .describe(
            'Regular post addresses for where information should be sent if needed.'
          ),
        id: zod
          .string()
          .optional()
          .describe('Unique identifier for the party (if available)'),
        name: zod
          .string()
          .optional()
          .describe('Legal name or representation of the organization.'),
        taxId: zod
          .object({
            code: zod
              .string()
              .min(1)
              .max(updateBillingProfileBodySupplierTaxIdCodeMaxOne)
              .describe(
                'TaxIdentificationCode is a normalized tax code shown on the original identity document.'
              )
              .optional()
              .describe(
                'Normalized tax code shown on the original identity document.'
              ),
          })
          .describe(
            'Identity stores the details required to identify an entity for tax purposes in a specific country.'
          )
          .optional()
          .describe(
            "The entity's legal ID code used for tax purposes. They may have\nother numbers, but we're only interested in those valid for tax purposes."
          ),
      })
      .describe('Party represents a person or business entity.')
      .describe(
        'The name and contact information for the supplier this billing profile represents'
      ),
    workflow: zod
      .object({
        collection: zod
          .object({
            alignment: zod
              .object({
                type: zod
                  .enum(['subscription'])
                  .describe('The type of alignment.'),
              })
              .describe(
                'BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items\ninto an invoice.'
              )
              .describe(
                'The alignment for collecting the pending line items into an invoice.\n\nDefaults to subscription, which means that we are to create a new invoice every time the\na subscription period starts (for in advance items) or ends (for in arrears items).'
              )
              .default(
                updateBillingProfileBodyWorkflowCollectionAlignmentDefault
              )
              .describe(
                'The alignment for collecting the pending line items into an invoice.'
              ),
            interval: zod
              .string()
              .default(
                updateBillingProfileBodyWorkflowCollectionIntervalDefault
              )
              .describe(
                'This grace period can be used to delay the collection of the pending line items specified in\nalignment.\n\nThis is useful, in case of multiple subscriptions having slightly different billing periods.'
              ),
          })
          .describe(
            'Workflow collection specifies how to collect the pending line items for an invoice'
          )
          .optional()
          .describe('The collection settings for this workflow'),
        invoicing: zod
          .object({
            autoAdvance: zod
              .boolean()
              .default(
                updateBillingProfileBodyWorkflowInvoicingAutoAdvanceDefault
              )
              .describe(
                'Whether to automatically issue the invoice after the draftPeriod has passed.'
              ),
            defaultTaxConfig: zod
              .object({
                behavior: zod
                  .enum(['inclusive', 'exclusive'])
                  .describe(
                    'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                  )
                  .optional()
                  .describe(
                    "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                  ),
                customInvoicing: zod
                  .object({
                    code: zod
                      .string()
                      .describe(
                        'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                      ),
                  })
                  .describe('Custom invoicing tax config.')
                  .optional()
                  .describe('Custom invoicing tax config.'),
                stripe: zod
                  .object({
                    code: zod
                      .string()
                      .regex(
                        updateBillingProfileBodyWorkflowInvoicingDefaultTaxConfigStripeCodeRegExp
                      )
                      .describe(
                        'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                      ),
                  })
                  .describe('The tax config for Stripe.')
                  .optional()
                  .describe('Stripe tax config.'),
              })
              .describe('Set of provider specific tax configs.')
              .optional()
              .describe('Default tax configuration to apply to the invoices.'),
            draftPeriod: zod
              .string()
              .default(
                updateBillingProfileBodyWorkflowInvoicingDraftPeriodDefault
              )
              .describe(
                'The period for the invoice to be kept in draft status for manual reviews.'
              ),
            dueAfter: zod
              .string()
              .default(updateBillingProfileBodyWorkflowInvoicingDueAfterDefault)
              .describe(
                "The period after which the invoice is due.\nWith some payment solutions it's only applicable for manual collection method."
              ),
            progressiveBilling: zod
              .boolean()
              .optional()
              .describe(
                'Should progressive billing be allowed for this workflow?'
              ),
          })
          .describe(
            'BillingWorkflowInvoicingSettings represents the invoice settings for a billing workflow'
          )
          .optional()
          .describe('The invoicing settings for this workflow'),
        payment: zod
          .object({
            collectionMethod: zod
              .enum(['charge_automatically', 'send_invoice'])
              .describe(
                'CollectionMethod specifies how the invoice should be collected (automatic vs manual)'
              )
              .default(
                updateBillingProfileBodyWorkflowPaymentCollectionMethodDefault
              )
              .describe('The payment method for the invoice.'),
          })
          .describe(
            'BillingWorkflowPaymentSettings represents the payment settings for a billing workflow'
          )
          .optional()
          .describe('The payment settings for this workflow'),
        tax: zod
          .object({
            enabled: zod
              .boolean()
              .default(updateBillingProfileBodyWorkflowTaxEnabledDefault)
              .describe(
                'Enable automatic tax calculation when tax is supported by the app.\nFor example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.'
              ),
            enforced: zod
              .boolean()
              .optional()
              .describe(
                'Enforce tax calculation when tax is supported by the app.\nWhen enabled, OpenMeter will not allow to create an invoice without tax calculation.\nEnforcement is different per apps, for example, Stripe app requires customer\nto have a tax location when starting a paid subscription.'
              ),
          })
          .describe(
            'BillingWorkflowTaxSettings represents the tax settings for a billing workflow'
          )
          .optional()
          .describe('The tax settings for this workflow'),
      })
      .describe(
        'BillingWorkflow represents the settings for a billing workflow.'
      )
      .describe('The billing workflow settings for this profile.'),
  })
  .describe(
    'BillingProfileReplaceUpdate represents the input for updating a billing profile\n\nThe apps field cannot be updated directly, if an app change is desired a new\nprofile should be created.'
  )

/**
 * Create a new customer.
 * @summary Create customer
 */
export const createCustomerBodyNameMax = 256
export const createCustomerBodyDescriptionMax = 1024
export const createCustomerBodyKeyMax = 256
export const createCustomerBodyUsageAttributionSubjectKeysMax = 1
export const createCustomerBodyCurrencyMinOne = 3

export const createCustomerBodyCurrencyMaxOne = 3

export const createCustomerBodyCurrencyRegExpOne = new RegExp('^[A-Z]{3}$')
export const createCustomerBodyBillingAddressCountryMinOne = 2

export const createCustomerBodyBillingAddressCountryMaxOne = 2

export const createCustomerBodyBillingAddressCountryRegExpOne = new RegExp(
  '^[A-Z]{2}$'
)

export const createCustomerBody = zod
  .object({
    billingAddress: zod
      .object({
        city: zod.string().optional().describe('City.'),
        country: zod
          .string()
          .min(createCustomerBodyBillingAddressCountryMinOne)
          .max(createCustomerBodyBillingAddressCountryMaxOne)
          .regex(createCustomerBodyBillingAddressCountryRegExpOne)
          .describe(
            '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
          )
          .optional()
          .describe(
            'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
          ),
        line1: zod.string().optional().describe('First line of the address.'),
        line2: zod.string().optional().describe('Second line of the address.'),
        phoneNumber: zod.string().optional().describe('Phone number.'),
        postalCode: zod.string().optional().describe('Postal code.'),
        state: zod.string().optional().describe('State or province.'),
      })
      .describe('Address')
      .optional()
      .describe(
        'The billing address of the customer.\nUsed for tax and invoicing.'
      ),
    currency: zod
      .string()
      .min(createCustomerBodyCurrencyMinOne)
      .max(createCustomerBodyCurrencyMaxOne)
      .regex(createCustomerBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .optional()
      .describe(
        'Currency of the customer.\nUsed for billing, tax and invoicing.'
      ),
    description: zod
      .string()
      .max(createCustomerBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    key: zod
      .string()
      .min(1)
      .max(createCustomerBodyKeyMax)
      .optional()
      .describe(
        'An optional unique key of the customer.\nUseful to reference the customer in external systems.\nFor example, your database ID.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createCustomerBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    primaryEmail: zod
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    usageAttribution: zod
      .object({
        subjectKeys: zod
          .array(zod.string())
          .min(1)
          .max(createCustomerBodyUsageAttributionSubjectKeysMax)
          .describe('The subjects that are attributed to the customer.'),
      })
      .describe(
        'Mapping to attribute metered usage to the customer.\nOne customer can have multiple subjects,\nbut one subject can only belong to one customer.'
      )
      .describe('Mapping to attribute metered usage to the customer'),
  })
  .describe('Resource create operation model.')

/**
 * List customers.
 * @summary List customers
 */
export const listCustomersQueryPageDefault = 1
export const listCustomersQueryPageSizeDefault = 100
export const listCustomersQueryPageSizeMax = 1000
export const listCustomersQueryIncludeDeletedDefault = false

export const listCustomersQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['subscriptions'])
        .describe(
          'CustomerExpand specifies the parts of the customer to expand in the list output.'
        )
    )
    .optional()
    .describe('What parts of the list output to expand in listings'),
  includeDeleted: zod
    .boolean()
    .optional()
    .describe('Include deleted customers.'),
  key: zod
    .string()
    .optional()
    .describe('Filter customers by key.\nCase-sensitive exact match.'),
  name: zod
    .string()
    .optional()
    .describe('Filter customers by name.\nCase-insensitive partial match.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'name', 'createdAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listCustomersQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listCustomersQueryPageSizeMax)
    .default(listCustomersQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  planKey: zod
    .string()
    .optional()
    .describe('Filter customers by the plan key of their susbcription.'),
  primaryEmail: zod
    .string()
    .optional()
    .describe(
      'Filter customers by primary email.\nCase-insensitive partial match.'
    ),
  subject: zod
    .string()
    .optional()
    .describe(
      'Filter customers by usage attribution subject.\nCase-insensitive partial match.'
    ),
})

/**
 * Get a customer by ID or key.
 * @summary Get customer
 */
export const getCustomerPathCustomerIdOrKeyMax = 64

export const getCustomerPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getCustomerParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(getCustomerPathCustomerIdOrKeyMax)
    .regex(getCustomerPathCustomerIdOrKeyRegExp),
})

export const getCustomerQueryParams = zod.object({
  expand: zod
    .array(
      zod
        .enum(['subscriptions'])
        .describe(
          'CustomerExpand specifies the parts of the customer to expand in the list output.'
        )
    )
    .optional()
    .describe('What parts of the customer output to expand'),
})

/**
 * Update a customer by ID.
 * @summary Update customer
 */
export const updateCustomerPathCustomerIdOrKeyMax = 64

export const updateCustomerPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateCustomerParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(updateCustomerPathCustomerIdOrKeyMax)
    .regex(updateCustomerPathCustomerIdOrKeyRegExp),
})

export const updateCustomerBodyNameMax = 256
export const updateCustomerBodyDescriptionMax = 1024
export const updateCustomerBodyKeyMax = 256
export const updateCustomerBodyUsageAttributionSubjectKeysMax = 1
export const updateCustomerBodyCurrencyMinOne = 3

export const updateCustomerBodyCurrencyMaxOne = 3

export const updateCustomerBodyCurrencyRegExpOne = new RegExp('^[A-Z]{3}$')
export const updateCustomerBodyBillingAddressCountryMinOne = 2

export const updateCustomerBodyBillingAddressCountryMaxOne = 2

export const updateCustomerBodyBillingAddressCountryRegExpOne = new RegExp(
  '^[A-Z]{2}$'
)

export const updateCustomerBody = zod
  .object({
    billingAddress: zod
      .object({
        city: zod.string().optional().describe('City.'),
        country: zod
          .string()
          .min(updateCustomerBodyBillingAddressCountryMinOne)
          .max(updateCustomerBodyBillingAddressCountryMaxOne)
          .regex(updateCustomerBodyBillingAddressCountryRegExpOne)
          .describe(
            '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
          )
          .optional()
          .describe(
            'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
          ),
        line1: zod.string().optional().describe('First line of the address.'),
        line2: zod.string().optional().describe('Second line of the address.'),
        phoneNumber: zod.string().optional().describe('Phone number.'),
        postalCode: zod.string().optional().describe('Postal code.'),
        state: zod.string().optional().describe('State or province.'),
      })
      .describe('Address')
      .optional()
      .describe(
        'The billing address of the customer.\nUsed for tax and invoicing.'
      ),
    currency: zod
      .string()
      .min(updateCustomerBodyCurrencyMinOne)
      .max(updateCustomerBodyCurrencyMaxOne)
      .regex(updateCustomerBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .optional()
      .describe(
        'Currency of the customer.\nUsed for billing, tax and invoicing.'
      ),
    description: zod
      .string()
      .max(updateCustomerBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    key: zod
      .string()
      .min(1)
      .max(updateCustomerBodyKeyMax)
      .optional()
      .describe(
        'An optional unique key of the customer.\nUseful to reference the customer in external systems.\nFor example, your database ID.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updateCustomerBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    primaryEmail: zod
      .string()
      .optional()
      .describe('The primary email address of the customer.'),
    usageAttribution: zod
      .object({
        subjectKeys: zod
          .array(zod.string())
          .min(1)
          .max(updateCustomerBodyUsageAttributionSubjectKeysMax)
          .describe('The subjects that are attributed to the customer.'),
      })
      .describe(
        'Mapping to attribute metered usage to the customer.\nOne customer can have multiple subjects,\nbut one subject can only belong to one customer.'
      )
      .describe('Mapping to attribute metered usage to the customer'),
  })
  .describe('Resource update operation model.')

/**
 * Delete a customer by ID.
 * @summary Delete customer
 */
export const deleteCustomerPathCustomerIdOrKeyMax = 64

export const deleteCustomerPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteCustomerParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(deleteCustomerPathCustomerIdOrKeyMax)
    .regex(deleteCustomerPathCustomerIdOrKeyRegExp),
})

/**
 * Get the overall access of a customer.
 * @summary Get customer access
 */
export const getCustomerAccessPathCustomerIdOrKeyMax = 64

export const getCustomerAccessPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getCustomerAccessParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(getCustomerAccessPathCustomerIdOrKeyMax)
    .regex(getCustomerAccessPathCustomerIdOrKeyRegExp),
})

/**
 * List customers app data.
 * @summary List customer app data
 */
export const listCustomerAppDataPathCustomerIdOrKeyMax = 64

export const listCustomerAppDataPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const listCustomerAppDataParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(listCustomerAppDataPathCustomerIdOrKeyMax)
    .regex(listCustomerAppDataPathCustomerIdOrKeyRegExp),
})

export const listCustomerAppDataQueryPageDefault = 1
export const listCustomerAppDataQueryPageSizeDefault = 100
export const listCustomerAppDataQueryPageSizeMax = 1000

export const listCustomerAppDataQueryParams = zod.object({
  page: zod
    .number()
    .min(1)
    .default(listCustomerAppDataQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listCustomerAppDataQueryPageSizeMax)
    .default(listCustomerAppDataQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  type: zod
    .enum(['stripe', 'sandbox', 'custom_invoicing'])
    .optional()
    .describe('Filter customer data by app type.'),
})

/**
 * Upsert customer app data.
 * @summary Upsert customer app data
 */
export const upsertCustomerAppDataPathCustomerIdOrKeyMax = 64

export const upsertCustomerAppDataPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const upsertCustomerAppDataParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(upsertCustomerAppDataPathCustomerIdOrKeyMax)
    .regex(upsertCustomerAppDataPathCustomerIdOrKeyRegExp),
})

export const upsertCustomerAppDataBodyIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const upsertCustomerAppDataBodyAppIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const upsertCustomerAppDataBodyAppNameMax = 256
export const upsertCustomerAppDataBodyAppDescriptionMax = 1024
export const upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyMax = 64

export const upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const upsertCustomerAppDataBodyIdRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const upsertCustomerAppDataBodyAppIdRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const upsertCustomerAppDataBodyAppNameMaxOne = 256
export const upsertCustomerAppDataBodyAppDescriptionMaxOne = 1024
export const upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyMaxOne = 64

export const upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const upsertCustomerAppDataBodyIdRegExpTwo = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const upsertCustomerAppDataBodyItem = zod
  .discriminatedUnion('type', [
    zod
      .object({
        id: zod
          .string()
          .regex(upsertCustomerAppDataBodyIdRegExp)
          .optional()
          .describe(
            'The app ID.\nIf not provided, it will use the global default for the app type.'
          ),
        stripeCustomerId: zod.string().describe('The Stripe customer ID.'),
        stripeDefaultPaymentMethodId: zod
          .string()
          .optional()
          .describe('The Stripe default payment method ID.'),
        type: zod.enum(['stripe']),
      })
      .describe('Stripe Customer App Data.'),
    zod
      .object({
        app: zod
          .object({
            createdAt: zod
              .date()
              .describe('Timestamp of when the resource was created.'),
            default: zod
              .boolean()
              .describe(
                'Default for the app type\nOnly one app of each type can be default.'
              ),
            deletedAt: zod
              .date()
              .optional()
              .describe(
                'Timestamp of when the resource was permanently deleted.'
              ),
            description: zod
              .string()
              .max(upsertCustomerAppDataBodyAppDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            id: zod
              .string()
              .regex(upsertCustomerAppDataBodyAppIdRegExp)
              .describe('A unique identifier for the resource.'),
            listing: zod
              .object({
                capabilities: zod
                  .array(
                    zod
                      .object({
                        description: zod
                          .string()
                          .describe('The capability description.'),
                        key: zod
                          .string()
                          .min(1)
                          .max(
                            upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyMax
                          )
                          .regex(
                            upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyRegExp
                          )
                          .describe('Key'),
                        name: zod.string().describe('The capability name.'),
                        type: zod
                          .enum([
                            'reportUsage',
                            'reportEvents',
                            'calculateTax',
                            'invoiceCustomers',
                            'collectPayments',
                          ])
                          .describe('App capability type.')
                          .describe('The capability type.'),
                      })
                      .describe(
                        "App capability.\n\nCapabilities only exist in config so they don't extend the Resource model."
                      )
                  )
                  .describe("The app's capabilities."),
                description: zod.string().describe("The app's description."),
                installMethods: zod
                  .array(
                    zod
                      .enum([
                        'with_oauth2',
                        'with_api_key',
                        'no_credentials_required',
                      ])
                      .describe('Install method of the application.')
                  )
                  .describe(
                    'Install methods.\n\nList of methods to install the app.'
                  ),
                name: zod.string().describe("The app's name."),
                type: zod
                  .enum(['stripe', 'sandbox', 'custom_invoicing'])
                  .describe('Type of the app.')
                  .describe("The app's type"),
              })
              .describe(
                "A marketplace listing.\nRepresent an available app in the app marketplace that can be installed to the organization.\n\nMarketplace apps only exist in config so they don't extend the Resource model."
              )
              .describe(
                'The marketplace listing that this installed app is based on.'
              ),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(upsertCustomerAppDataBodyAppNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            status: zod
              .enum(['ready', 'unauthorized'])
              .describe('App installed status.')
              .describe('Status of the app connection.'),
            type: zod.enum(['sandbox']),
            updatedAt: zod
              .date()
              .describe('Timestamp of when the resource was last updated.'),
          })
          .describe(
            'Sandbox app can be used for testing OpenMeter features.\n\nThe app is not creating anything in external systems, thus it is safe to use for\nverifying OpenMeter features.'
          )
          .optional()
          .describe('The installed sandbox app this data belongs to.'),
        id: zod
          .string()
          .regex(upsertCustomerAppDataBodyIdRegExpOne)
          .optional()
          .describe(
            'The app ID.\nIf not provided, it will use the global default for the app type.'
          ),
        type: zod.enum(['sandbox']),
      })
      .describe('Sandbox Customer App Data.'),
    zod
      .object({
        app: zod
          .object({
            createdAt: zod
              .date()
              .describe('Timestamp of when the resource was created.'),
            default: zod
              .boolean()
              .describe(
                'Default for the app type\nOnly one app of each type can be default.'
              ),
            deletedAt: zod
              .date()
              .optional()
              .describe(
                'Timestamp of when the resource was permanently deleted.'
              ),
            description: zod
              .string()
              .max(upsertCustomerAppDataBodyAppDescriptionMaxOne)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            enableDraftSyncHook: zod
              .boolean()
              .describe(
                'Enable draft.sync hook.\n\nIf the hook is not enabled, the invoice will be progressed to the next state automatically.'
              ),
            enableIssuingSyncHook: zod
              .boolean()
              .describe(
                'Enable issuing.sync hook.\n\nIf the hook is not enabled, the invoice will be progressed to the next state automatically.'
              ),
            id: zod
              .string()
              .regex(upsertCustomerAppDataBodyAppIdRegExpOne)
              .describe('A unique identifier for the resource.'),
            listing: zod
              .object({
                capabilities: zod
                  .array(
                    zod
                      .object({
                        description: zod
                          .string()
                          .describe('The capability description.'),
                        key: zod
                          .string()
                          .min(1)
                          .max(
                            upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyMaxOne
                          )
                          .regex(
                            upsertCustomerAppDataBodyAppListingCapabilitiesItemKeyRegExpOne
                          )
                          .describe('Key'),
                        name: zod.string().describe('The capability name.'),
                        type: zod
                          .enum([
                            'reportUsage',
                            'reportEvents',
                            'calculateTax',
                            'invoiceCustomers',
                            'collectPayments',
                          ])
                          .describe('App capability type.')
                          .describe('The capability type.'),
                      })
                      .describe(
                        "App capability.\n\nCapabilities only exist in config so they don't extend the Resource model."
                      )
                  )
                  .describe("The app's capabilities."),
                description: zod.string().describe("The app's description."),
                installMethods: zod
                  .array(
                    zod
                      .enum([
                        'with_oauth2',
                        'with_api_key',
                        'no_credentials_required',
                      ])
                      .describe('Install method of the application.')
                  )
                  .describe(
                    'Install methods.\n\nList of methods to install the app.'
                  ),
                name: zod.string().describe("The app's name."),
                type: zod
                  .enum(['stripe', 'sandbox', 'custom_invoicing'])
                  .describe('Type of the app.')
                  .describe("The app's type"),
              })
              .describe(
                "A marketplace listing.\nRepresent an available app in the app marketplace that can be installed to the organization.\n\nMarketplace apps only exist in config so they don't extend the Resource model."
              )
              .describe(
                'The marketplace listing that this installed app is based on.'
              ),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(upsertCustomerAppDataBodyAppNameMaxOne)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            status: zod
              .enum(['ready', 'unauthorized'])
              .describe('App installed status.')
              .describe('Status of the app connection.'),
            type: zod.enum(['custom_invoicing']),
            updatedAt: zod
              .date()
              .describe('Timestamp of when the resource was last updated.'),
          })
          .describe(
            'Custom Invoicing app can be used for interface with any invoicing or payment system.\n\nThis app provides ways to manipulate invoices and payments, however the integration\nmust rely on Notifications API to get notified about invoice changes.'
          )
          .optional()
          .describe('The installed custom invoicing app this data belongs to.'),
        id: zod
          .string()
          .regex(upsertCustomerAppDataBodyIdRegExpTwo)
          .optional()
          .describe(
            'The app ID.\nIf not provided, it will use the global default for the app type.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Metadata to be used by the custom invoicing provider.'),
        type: zod.enum(['custom_invoicing']),
      })
      .describe('Custom Invoicing Customer App Data.'),
  ])
  .describe(
    'CustomerAppData\nStores the app specific data for the customer.\nOne of: stripe, sandbox, custom_invoicing'
  )
export const upsertCustomerAppDataBody = zod.array(
  upsertCustomerAppDataBodyItem
)

/**
 * Delete customer app data.
 * @summary Delete customer app data
 */
export const deleteCustomerAppDataPathCustomerIdOrKeyMax = 64

export const deleteCustomerAppDataPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const deleteCustomerAppDataPathAppIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteCustomerAppDataParams = zod.object({
  appId: zod.string().regex(deleteCustomerAppDataPathAppIdRegExp),
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(deleteCustomerAppDataPathCustomerIdOrKeyMax)
    .regex(deleteCustomerAppDataPathCustomerIdOrKeyRegExp),
})

/**
 * Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
 * @summary Get entitlement value
 */
export const getCustomerEntitlementValuePathCustomerIdOrKeyMax = 64

export const getCustomerEntitlementValuePathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const getCustomerEntitlementValuePathFeatureKeyMax = 64

export const getCustomerEntitlementValuePathFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)

export const getCustomerEntitlementValueParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(getCustomerEntitlementValuePathCustomerIdOrKeyMax)
    .regex(getCustomerEntitlementValuePathCustomerIdOrKeyRegExp),
  featureKey: zod
    .string()
    .min(1)
    .max(getCustomerEntitlementValuePathFeatureKeyMax)
    .regex(getCustomerEntitlementValuePathFeatureKeyRegExp),
})

export const getCustomerEntitlementValueQueryParams = zod.object({
  time: zod.date().optional(),
})

/**
 * Lists all subscriptions for a customer.
 * @summary List customer subscriptions
 */
export const listCustomerSubscriptionsPathCustomerIdOrKeyMax = 64

export const listCustomerSubscriptionsPathCustomerIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const listCustomerSubscriptionsParams = zod.object({
  customerIdOrKey: zod
    .string()
    .min(1)
    .max(listCustomerSubscriptionsPathCustomerIdOrKeyMax)
    .regex(listCustomerSubscriptionsPathCustomerIdOrKeyRegExp),
})

export const listCustomerSubscriptionsQueryPageDefault = 1
export const listCustomerSubscriptionsQueryPageSizeDefault = 100
export const listCustomerSubscriptionsQueryPageSizeMax = 1000

export const listCustomerSubscriptionsQueryParams = zod.object({
  page: zod
    .number()
    .min(1)
    .default(listCustomerSubscriptionsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listCustomerSubscriptionsQueryPageSizeMax)
    .default(listCustomerSubscriptionsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * List all entitlements for all the subjects and features. This endpoint is intended for administrative purposes only.
To fetch the entitlements of a specific subject please use the /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint.
If page is provided that takes precedence and the paginated response is returned.
 * @summary List all entitlements
 */
export const listEntitlementsQueryExcludeInactiveDefault = false
export const listEntitlementsQueryPageDefault = 1
export const listEntitlementsQueryPageSizeDefault = 100
export const listEntitlementsQueryPageSizeMax = 1000
export const listEntitlementsQueryOffsetDefault = 0
export const listEntitlementsQueryOffsetMin = 0
export const listEntitlementsQueryLimitDefault = 100
export const listEntitlementsQueryLimitMax = 1000

export const listEntitlementsQueryParams = zod.object({
  entitlementType: zod
    .array(
      zod
        .enum(['metered', 'boolean', 'static'])
        .describe('Type of the entitlement.')
    )
    .optional()
    .describe(
      'Filtering by multiple entitlement types.\n\nUsage: `?entitlementType=metered&entitlementType=boolean`'
    ),
  excludeInactive: zod
    .boolean()
    .optional()
    .describe(
      'Exclude inactive entitlements in the response (those scheduled for later or earlier)'
    ),
  feature: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple features.\n\nUsage: `?feature=feature-1&feature=feature-2`'
    ),
  limit: zod
    .number()
    .min(1)
    .max(listEntitlementsQueryLimitMax)
    .default(listEntitlementsQueryLimitDefault)
    .describe('Number of items to return.\n\nDefault is 100.'),
  offset: zod
    .number()
    .min(listEntitlementsQueryOffsetMin)
    .optional()
    .describe('Number of items to skip.\n\nDefault is 0.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listEntitlementsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listEntitlementsQueryPageSizeMax)
    .default(listEntitlementsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  subject: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple subjects.\n\nUsage: `?subject=customer-1&subject=customer-2`'
    ),
})

/**
 * Get entitlement by id.
 * @summary Get entitlement by id
 */
export const getEntitlementByIdPathEntitlementIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getEntitlementByIdParams = zod.object({
  entitlementId: zod.string().regex(getEntitlementByIdPathEntitlementIdRegExp),
})

/**
 * List ingested events within a time range.

If the from query param is not provided it defaults to last 72 hours.
 * @summary List ingested events
 */
export const listEventsQueryClientIdMax = 36
export const listEventsQueryLimitDefault = 100
export const listEventsQueryLimitMax = 100

export const listEventsQueryParams = zod.object({
  clientId: zod
    .string()
    .min(1)
    .max(listEventsQueryClientIdMax)
    .optional()
    .describe('Client ID\nUseful to track progress of a query.'),
  from: zod
    .date()
    .optional()
    .describe('Start date-time in RFC 3339 format.\n\nInclusive.'),
  id: zod.string().optional().describe('The event ID.\n\nAccepts partial ID.'),
  ingestedAtFrom: zod
    .date()
    .optional()
    .describe('Start date-time in RFC 3339 format.\n\nInclusive.'),
  ingestedAtTo: zod
    .date()
    .optional()
    .describe('End date-time in RFC 3339 format.\n\nInclusive.'),
  limit: zod
    .number()
    .min(1)
    .max(listEventsQueryLimitMax)
    .default(listEventsQueryLimitDefault)
    .describe('Number of events to return.'),
  subject: zod
    .string()
    .optional()
    .describe('The event subject.\n\nAccepts partial subject.'),
  to: zod
    .date()
    .optional()
    .describe('End date-time in RFC 3339 format.\n\nInclusive.'),
})

/**
 * Ingests an event or batch of events following the CloudEvents specification.
 * @summary Ingest events
 */
export const ingestEventsBodySpecversionDefault = '1.0'
export const ingestEventsBodyItemSpecversionDefault = '1.0'

export const ingestEventsBody = zod
  .object({
    data: zod
      .record(zod.string(), zod.any())
      .nullish()
      .describe(
        'The event payload.\nOptional, if present it must be a JSON object.'
      ),
    datacontenttype: zod
      .enum(['application/json'])
      .nullish()
      .describe(
        'Content type of the CloudEvents data value. Only the value \"application/json\" is allowed over HTTP.'
      ),
    dataschema: zod
      .string()
      .url()
      .min(1)
      .nullish()
      .describe('Identifies the schema that data adheres to.'),
    id: zod.string().min(1).describe('Identifies the event.'),
    source: zod
      .string()
      .min(1)
      .describe('Identifies the context in which an event happened.'),
    specversion: zod
      .string()
      .min(1)
      .describe(
        'The version of the CloudEvents specification which the event uses.'
      ),
    subject: zod
      .string()
      .min(1)
      .describe(
        'Describes the subject of the event in the context of the event producer (identified by source).'
      ),
    time: zod
      .date()
      .nullish()
      .describe(
        'Timestamp of when the occurrence happened. Must adhere to RFC 3339.'
      ),
    type: zod
      .string()
      .min(1)
      .describe(
        'Contains a value describing the type of event related to the originating occurrence.'
      ),
  })
  .describe(
    'CloudEvents Specification JSON Schema\n\nOptional properties are nullable according to the CloudEvents specification:\nOPTIONAL not omitted attributes MAY be represented as a null JSON value.'
  )
  .or(
    zod.array(
      zod
        .object({
          data: zod
            .record(zod.string(), zod.any())
            .nullish()
            .describe(
              'The event payload.\nOptional, if present it must be a JSON object.'
            ),
          datacontenttype: zod
            .enum(['application/json'])
            .nullish()
            .describe(
              'Content type of the CloudEvents data value. Only the value \"application/json\" is allowed over HTTP.'
            ),
          dataschema: zod
            .string()
            .url()
            .min(1)
            .nullish()
            .describe('Identifies the schema that data adheres to.'),
          id: zod.string().min(1).describe('Identifies the event.'),
          source: zod
            .string()
            .min(1)
            .describe('Identifies the context in which an event happened.'),
          specversion: zod
            .string()
            .min(1)
            .describe(
              'The version of the CloudEvents specification which the event uses.'
            ),
          subject: zod
            .string()
            .min(1)
            .describe(
              'Describes the subject of the event in the context of the event producer (identified by source).'
            ),
          time: zod
            .date()
            .nullish()
            .describe(
              'Timestamp of when the occurrence happened. Must adhere to RFC 3339.'
            ),
          type: zod
            .string()
            .min(1)
            .describe(
              'Contains a value describing the type of event related to the originating occurrence.'
            ),
        })
        .describe(
          'CloudEvents Specification JSON Schema\n\nOptional properties are nullable according to the CloudEvents specification:\nOPTIONAL not omitted attributes MAY be represented as a null JSON value.'
        )
    )
  )
  .describe(
    'The body of the events request.\nEither a single event or a batch of events.'
  )

/**
 * List features.
 * @summary List features
 */
export const listFeaturesQueryIncludeArchivedDefault = false
export const listFeaturesQueryPageDefault = 1
export const listFeaturesQueryPageSizeDefault = 100
export const listFeaturesQueryPageSizeMax = 1000
export const listFeaturesQueryOffsetDefault = 0
export const listFeaturesQueryOffsetMin = 0
export const listFeaturesQueryLimitDefault = 100
export const listFeaturesQueryLimitMax = 1000

export const listFeaturesQueryParams = zod.object({
  includeArchived: zod
    .boolean()
    .optional()
    .describe('Filter by meterGroupByFilters'),
  limit: zod
    .number()
    .min(1)
    .max(listFeaturesQueryLimitMax)
    .default(listFeaturesQueryLimitDefault)
    .describe('Number of items to return.\n\nDefault is 100.'),
  meterSlug: zod.array(zod.string()).optional().describe('Filter by meterSlug'),
  offset: zod
    .number()
    .min(listFeaturesQueryOffsetMin)
    .optional()
    .describe('Number of items to skip.\n\nDefault is 0.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'key', 'name', 'createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listFeaturesQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listFeaturesQueryPageSizeMax)
    .default(listFeaturesQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Features are either metered or static. A feature is metered if meterSlug is provided at creation.
For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields.
Only meters with SUM and COUNT aggregation are supported for features.
Features cannot be updated later, only archived.
 * @summary Create feature
 */
export const createFeatureBodyKeyMax = 64

export const createFeatureBodyKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createFeatureBodyMeterSlugMax = 64

export const createFeatureBodyMeterSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)

export const createFeatureBody = zod
  .object({
    key: zod
      .string()
      .min(1)
      .max(createFeatureBodyKeyMax)
      .regex(createFeatureBodyKeyRegExp)
      .describe(
        'A key is a unique string that is used to identify a resource.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional(),
    meterGroupByFilters: zod
      .record(zod.string(), zod.string())
      .optional()
      .describe(
        'Optional meter group by filters.\nUseful if the meter scope is broader than what feature tracks.\nExample scenario would be a meter tracking all token use with groupBy fields for the model,\nthen the feature could filter for model=gpt-4.'
      ),
    meterSlug: zod
      .string()
      .min(1)
      .max(createFeatureBodyMeterSlugMax)
      .regex(createFeatureBodyMeterSlugRegExp)
      .optional()
      .describe(
        'A key is a unique string that is used to identify a resource.'
      ),
    name: zod.string(),
  })
  .describe(
    'Represents a feature that can be enabled or disabled for a plan.\nUsed both for product catalog and entitlements.'
  )

/**
 * Get a feature by ID.
 * @summary Get feature
 */
export const getFeatureParams = zod.object({
  featureId: zod.string(),
})

/**
 * Archive a feature by ID.

Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.
 * @summary Delete feature
 */
export const deleteFeatureParams = zod.object({
  featureId: zod.string(),
})

/**
 * List all grants for all the subjects and entitlements. This endpoint is intended for administrative purposes only.
To fetch the grants of a specific entitlement please use the /api/v1/subjects/{subjectKeyOrID}/entitlements/{entitlementOrFeatureID}/grants endpoint.
If page is provided that takes precedence and the paginated response is returned.
 * @summary List grants
 */
export const listGrantsQueryIncludeDeletedDefault = false
export const listGrantsQueryPageDefault = 1
export const listGrantsQueryPageSizeDefault = 100
export const listGrantsQueryPageSizeMax = 1000
export const listGrantsQueryOffsetDefault = 0
export const listGrantsQueryOffsetMin = 0
export const listGrantsQueryLimitDefault = 100
export const listGrantsQueryLimitMax = 1000

export const listGrantsQueryParams = zod.object({
  feature: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple features.\n\nUsage: `?feature=feature-1&feature=feature-2`'
    ),
  includeDeleted: zod.boolean().optional().describe('Include deleted'),
  limit: zod
    .number()
    .min(1)
    .max(listGrantsQueryLimitMax)
    .default(listGrantsQueryLimitDefault)
    .describe('Number of items to return.\n\nDefault is 100.'),
  offset: zod
    .number()
    .min(listGrantsQueryOffsetMin)
    .optional()
    .describe('Number of items to skip.\n\nDefault is 0.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listGrantsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listGrantsQueryPageSizeMax)
    .default(listGrantsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  subject: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple subjects.\n\nUsage: `?subject=customer-1&subject=customer-2`'
    ),
})

/**
 * Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant.
For example, if you have a single grant for your metered entitlement with an initial amount of 100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will not be affected.
 * @summary Void grant
 */
export const voidGrantParams = zod.object({
  grantId: zod.string(),
})

/**
 * Get progress
 * @summary Get progress
 */
export const getProgressParams = zod.object({
  id: zod.string(),
})

/**
 * List available apps of the app marketplace.
 * @summary List available apps
 */
export const listMarketplaceListingsQueryPageDefault = 1
export const listMarketplaceListingsQueryPageSizeDefault = 100
export const listMarketplaceListingsQueryPageSizeMax = 1000

export const listMarketplaceListingsQueryParams = zod.object({
  page: zod
    .number()
    .min(1)
    .default(listMarketplaceListingsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listMarketplaceListingsQueryPageSizeMax)
    .default(listMarketplaceListingsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Get a marketplace listing by type.
 * @summary Get app details by type
 */
export const getMarketplaceListingParams = zod.object({
  type: zod.enum(['stripe', 'sandbox', 'custom_invoicing']),
})

/**
 * Install an app from the marketplace.
 * @summary Install app
 */
export const marketplaceAppInstallParams = zod.object({
  type: zod
    .enum(['stripe', 'sandbox', 'custom_invoicing'])
    .describe('The type of the app to install.'),
})

export const marketplaceAppInstallBody = zod.object({
  name: zod
    .string()
    .optional()
    .describe(
      "Name of the application to install.\n\nIf not set defaults to the marketplace item's description."
    ),
})

/**
 * Install an marketplace app via API Key.
 * @summary Install app via API key
 */
export const marketplaceAppAPIKeyInstallParams = zod.object({
  type: zod
    .enum(['stripe', 'sandbox', 'custom_invoicing'])
    .describe('The type of the app to install.'),
})

export const marketplaceAppAPIKeyInstallBody = zod.object({
  apiKey: zod
    .string()
    .describe(
      'The API key for the provider.\nFor example, the Stripe API key.'
    ),
  name: zod
    .string()
    .optional()
    .describe(
      "Name of the application to install.\n\nIf not set defaults to the marketplace item's description."
    ),
})

/**
 * Install an app via OAuth.
Returns a URL to start the OAuth 2.0 flow.
 * @summary Get OAuth2 install URL
 */
export const marketplaceOAuth2InstallGetURLParams = zod.object({
  type: zod.enum(['stripe', 'sandbox', 'custom_invoicing']),
})

/**
 * Authorize OAuth2 code.
Verifies the OAuth code and exchanges it for a token and refresh token
 * @summary Install app via OAuth2
 */
export const marketplaceOAuth2InstallAuthorizeParams = zod.object({
  type: zod
    .enum(['stripe', 'sandbox', 'custom_invoicing'])
    .describe('The type of the app to install.'),
})

export const marketplaceOAuth2InstallAuthorizeQueryParams = zod.object({
  code: zod
    .string()
    .optional()
    .describe(
      'Authorization code which the client will later exchange for an access token.\nRequired with the success response.'
    ),
  error: zod
    .enum([
      'invalid_request',
      'unauthorized_client',
      'access_denied',
      'unsupported_response_type',
      'invalid_scope',
      'server_error',
      'temporarily_unavailable',
    ])
    .optional()
    .describe('Error code.\nRequired with the error response.'),
  error_description: zod
    .string()
    .optional()
    .describe(
      'Optional human-readable text providing additional information,\nused to assist the client developer in understanding the error that occurred.'
    ),
  error_uri: zod
    .string()
    .optional()
    .describe(
      'Optional uri identifying a human-readable web page with\ninformation about the error, used to provide the client\ndeveloper with additional information about the error'
    ),
  state: zod
    .string()
    .optional()
    .describe(
      'Required if the \"state\" parameter was present in the client authorization request.\nThe exact value received from the client:\n\nUnique, randomly generated, opaque, and non-guessable string that is sent\nwhen starting an authentication request and validated when processing the response.'
    ),
})

/**
 * List meters.
 * @summary List meters
 */
export const listMetersQueryPageDefault = 1
export const listMetersQueryPageSizeDefault = 100
export const listMetersQueryPageSizeMax = 1000
export const listMetersQueryIncludeDeletedDefault = false

export const listMetersQueryParams = zod.object({
  includeDeleted: zod.boolean().optional().describe('Include deleted meters.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['key', 'name', 'aggregation', 'createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listMetersQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listMetersQueryPageSizeMax)
    .default(listMetersQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Create a meter.
 * @summary Create meter
 */
export const createMeterBodyDescriptionMax = 1024
export const createMeterBodyNameMax = 256
export const createMeterBodySlugMax = 64

export const createMeterBodySlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)

export const createMeterBody = zod
  .object({
    aggregation: zod
      .enum(['SUM', 'COUNT', 'UNIQUE_COUNT', 'AVG', 'MIN', 'MAX', 'LATEST'])
      .describe('The aggregation type to use for the meter.')
      .describe('The aggregation type to use for the meter.'),
    description: zod
      .string()
      .max(createMeterBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    eventFrom: zod
      .date()
      .optional()
      .describe(
        'The date since the meter should include events.\nUseful to skip old events.\nIf not specified, all historical events are included.'
      ),
    eventType: zod.string().min(1).describe('The event type to aggregate.'),
    groupBy: zod
      .record(zod.string(), zod.string())
      .optional()
      .describe(
        'Named JSONPath expressions to extract the group by values from the event data.\n\nKeys must be unique and consist only alphanumeric and underscore characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createMeterBodyNameMax)
      .optional()
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.\nDefaults to the slug if not specified.'
      ),
    slug: zod
      .string()
      .min(1)
      .max(createMeterBodySlugMax)
      .regex(createMeterBodySlugRegExp)
      .describe(
        'A unique, human-readable identifier for the meter.\nMust consist only alphanumeric and underscore characters.'
      ),
    valueProperty: zod
      .string()
      .min(1)
      .optional()
      .describe(
        "JSONPath expression to extract the value from the ingested event's data property.\n\nThe ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.\n\nFor UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored."
      ),
  })
  .describe('A meter create model.')

/**
 * Get a meter by ID or slug.
 * @summary Get meter
 */
export const getMeterPathMeterIdOrSlugMax = 64

export const getMeterPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getMeterParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(getMeterPathMeterIdOrSlugMax)
    .regex(getMeterPathMeterIdOrSlugRegExp),
})

/**
 * Update a meter.
 * @summary Update meter
 */
export const updateMeterPathMeterIdOrSlugMax = 64

export const updateMeterPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateMeterParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(updateMeterPathMeterIdOrSlugMax)
    .regex(updateMeterPathMeterIdOrSlugRegExp),
})

export const updateMeterBodyDescriptionMax = 1024
export const updateMeterBodyNameMax = 256

export const updateMeterBody = zod
  .object({
    description: zod
      .string()
      .max(updateMeterBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    groupBy: zod
      .record(zod.string(), zod.string())
      .optional()
      .describe(
        'Named JSONPath expressions to extract the group by values from the event data.\n\nKeys must be unique and consist only alphanumeric and underscore characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updateMeterBodyNameMax)
      .optional()
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.\nDefaults to the slug if not specified.'
      ),
  })
  .describe(
    'A meter update model.\n\nOnly the properties that can be updated are included.\nFor example, the slug and aggregation cannot be updated.'
  )

/**
 * Delete a meter.
 * @summary Delete meter
 */
export const deleteMeterPathMeterIdOrSlugMax = 64

export const deleteMeterPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteMeterParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(deleteMeterPathMeterIdOrSlugMax)
    .regex(deleteMeterPathMeterIdOrSlugRegExp),
})

/**
 * Query meter for usage.
 * @summary Query meter
 */
export const queryMeterPathMeterIdOrSlugMax = 64

export const queryMeterPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const queryMeterParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(queryMeterPathMeterIdOrSlugMax)
    .regex(queryMeterPathMeterIdOrSlugRegExp),
})

export const queryMeterQueryClientIdMax = 36
export const queryMeterQueryWindowTimeZoneDefault = 'UTC'

export const queryMeterQueryParams = zod.object({
  clientId: zod
    .string()
    .min(1)
    .max(queryMeterQueryClientIdMax)
    .optional()
    .describe('Client ID\nUseful to track progress of a query.'),
  filterGroupBy: zod
    .record(zod.string(), zod.string())
    .optional()
    .describe(
      'Simple filter for group bys with exact match.\n\nFor example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo'
    ),
  from: zod
    .date()
    .optional()
    .describe(
      'Start date-time in RFC 3339 format.\n\nInclusive.\n\nFor example: ?from=2025-01-01T00%3A00%3A00.000Z'
    ),
  groupBy: zod
    .array(zod.string())
    .optional()
    .describe(
      'If not specified a single aggregate will be returned for each subject and time window.\n`subject` is a reserved group by value.\n\nFor example: ?groupBy=subject&groupBy=model'
    ),
  subject: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple subjects.\n\nFor example: ?subject=customer-1&subject=customer-2'
    ),
  to: zod
    .date()
    .optional()
    .describe(
      'End date-time in RFC 3339 format.\n\nInclusive.\n\nFor example: ?to=2025-02-01T00%3A00%3A00.000Z'
    ),
  windowSize: zod
    .enum(['MINUTE', 'HOUR', 'DAY'])
    .optional()
    .describe(
      'If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.\n\nFor example: ?windowSize=DAY'
    ),
  windowTimeZone: zod
    .string()
    .default(queryMeterQueryWindowTimeZoneDefault)
    .describe(
      'The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).\nIf not specified, the UTC timezone will be used.\n\nFor example: ?windowTimeZone=UTC'
    ),
})

/**
 * @summary Query meter
 */
export const queryMeterPostPathMeterIdOrSlugMax = 64

export const queryMeterPostPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const queryMeterPostParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(queryMeterPostPathMeterIdOrSlugMax)
    .regex(queryMeterPostPathMeterIdOrSlugRegExp),
})

export const queryMeterPostBodyClientIdMax = 36
export const queryMeterPostBodyWindowTimeZoneDefault = 'UTC'
export const queryMeterPostBodySubjectMax = 100
export const queryMeterPostBodyGroupByMax = 100

export const queryMeterPostBody = zod
  .object({
    clientId: zod
      .string()
      .min(1)
      .max(queryMeterPostBodyClientIdMax)
      .optional()
      .describe('Client ID\nUseful to track progress of a query.'),
    filterGroupBy: zod
      .record(zod.string(), zod.array(zod.string()))
      .optional()
      .describe('Simple filter for group bys with exact match.'),
    from: zod
      .date()
      .optional()
      .describe('Start date-time in RFC 3339 format.\n\nInclusive.'),
    groupBy: zod
      .array(zod.string())
      .max(queryMeterPostBodyGroupByMax)
      .optional()
      .describe(
        'If not specified a single aggregate will be returned for each subject and time window.\n`subject` is a reserved group by value.'
      ),
    subject: zod
      .array(zod.string())
      .max(queryMeterPostBodySubjectMax)
      .optional()
      .describe('Filtering by multiple subjects.'),
    to: zod
      .date()
      .optional()
      .describe('End date-time in RFC 3339 format.\n\nInclusive.'),
    windowSize: zod
      .enum(['MINUTE', 'HOUR', 'DAY'])
      .describe('Aggregation window size.')
      .optional()
      .describe(
        'If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.'
      ),
    windowTimeZone: zod
      .string()
      .default(queryMeterPostBodyWindowTimeZoneDefault)
      .describe(
        'The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).\nIf not specified, the UTC timezone will be used.'
      ),
  })
  .describe('A meter query request.')

/**
 * List subjects for a meter.
 * @summary List meter subjects
 */
export const listMeterSubjectsPathMeterIdOrSlugMax = 64

export const listMeterSubjectsPathMeterIdOrSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const listMeterSubjectsParams = zod.object({
  meterIdOrSlug: zod
    .string()
    .min(1)
    .max(listMeterSubjectsPathMeterIdOrSlugMax)
    .regex(listMeterSubjectsPathMeterIdOrSlugRegExp),
})

/**
 * List all notification channels.
 * @summary List notification channels
 */
export const listNotificationChannelsQueryIncludeDeletedDefault = false
export const listNotificationChannelsQueryIncludeDisabledDefault = false
export const listNotificationChannelsQueryPageDefault = 1
export const listNotificationChannelsQueryPageSizeDefault = 100
export const listNotificationChannelsQueryPageSizeMax = 1000

export const listNotificationChannelsQueryParams = zod.object({
  includeDeleted: zod
    .boolean()
    .optional()
    .describe(
      'Include deleted notification channels in response.\n\nUsage: `?includeDeleted=true`'
    ),
  includeDisabled: zod
    .boolean()
    .optional()
    .describe(
      'Include disabled notification channels in response.\n\nUsage: `?includeDisabled=false`'
    ),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'type', 'createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listNotificationChannelsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listNotificationChannelsQueryPageSizeMax)
    .default(listNotificationChannelsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Create a new notification channel.
 * @summary Create a notification channel
 */
export const createNotificationChannelBodyDisabledDefault = false
export const createNotificationChannelBodySigningSecretRegExp = new RegExp(
  '^(whsec_)?[a-zA-Z0-9+/=]{32,100}$'
)

export const createNotificationChannelBody = zod
  .object({
    customHeaders: zod
      .record(zod.string(), zod.string())
      .optional()
      .describe('Custom HTTP headers sent as part of the webhook request.'),
    disabled: zod
      .boolean()
      .optional()
      .describe('Whether the channel is disabled or not.'),
    name: zod.string().describe('User friendly name of the channel.'),
    signingSecret: zod
      .string()
      .regex(createNotificationChannelBodySigningSecretRegExp)
      .optional()
      .describe(
        'Signing secret used for webhook request validation on the receiving end.\n\nFormat: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24'
      ),
    type: zod.enum(['WEBHOOK']).describe('Notification channel type.'),
    url: zod.string().describe('Webhook URL where the notification is sent.'),
  })
  .describe(
    'Request with input parameters for creating new notification channel with webhook type.'
  )
  .describe(
    'Union type for requests creating new notification channel with certain type.'
  )

/**
 * Update notification channel.
 * @summary Update a notification channel
 */
export const updateNotificationChannelPathChannelIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateNotificationChannelParams = zod.object({
  channelId: zod.string().regex(updateNotificationChannelPathChannelIdRegExp),
})

export const updateNotificationChannelBodyDisabledDefault = false
export const updateNotificationChannelBodySigningSecretRegExp = new RegExp(
  '^(whsec_)?[a-zA-Z0-9+/=]{32,100}$'
)

export const updateNotificationChannelBody = zod
  .object({
    customHeaders: zod
      .record(zod.string(), zod.string())
      .optional()
      .describe('Custom HTTP headers sent as part of the webhook request.'),
    disabled: zod
      .boolean()
      .optional()
      .describe('Whether the channel is disabled or not.'),
    name: zod.string().describe('User friendly name of the channel.'),
    signingSecret: zod
      .string()
      .regex(updateNotificationChannelBodySigningSecretRegExp)
      .optional()
      .describe(
        'Signing secret used for webhook request validation on the receiving end.\n\nFormat: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24'
      ),
    type: zod.enum(['WEBHOOK']).describe('Notification channel type.'),
    url: zod.string().describe('Webhook URL where the notification is sent.'),
  })
  .describe(
    'Request with input parameters for creating new notification channel with webhook type.'
  )
  .describe(
    'Union type for requests creating new notification channel with certain type.'
  )

/**
 * Get a notification channel by id.
 * @summary Get notification channel
 */
export const getNotificationChannelPathChannelIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getNotificationChannelParams = zod.object({
  channelId: zod.string().regex(getNotificationChannelPathChannelIdRegExp),
})

/**
 * Soft delete notification channel by id.

Once a notification channel is deleted it cannot be undeleted.
 * @summary Delete a notification channel
 */
export const deleteNotificationChannelPathChannelIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteNotificationChannelParams = zod.object({
  channelId: zod.string().regex(deleteNotificationChannelPathChannelIdRegExp),
})

/**
 * List all notification events.
 * @summary List notification events
 */
export const listNotificationEventsQueryRuleItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listNotificationEventsQueryChannelItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listNotificationEventsQueryPageDefault = 1
export const listNotificationEventsQueryPageSizeDefault = 100
export const listNotificationEventsQueryPageSizeMax = 1000

export const listNotificationEventsQueryParams = zod.object({
  channel: zod
    .array(
      zod
        .string()
        .regex(listNotificationEventsQueryChannelItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe(
      'Filtering by multiple channel ids.\n\nUsage: `?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J`'
    ),
  feature: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple feature ids or keys.\n\nUsage: `?feature=feature-1&feature=feature-2`'
    ),
  from: zod
    .date()
    .optional()
    .describe('Start date-time in RFC 3339 format.\nInclusive.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'createdAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listNotificationEventsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listNotificationEventsQueryPageSizeMax)
    .default(listNotificationEventsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  rule: zod
    .array(
      zod
        .string()
        .regex(listNotificationEventsQueryRuleItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe(
      'Filtering by multiple rule ids.\n\nUsage: `?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5`'
    ),
  subject: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple subject ids or keys.\n\nUsage: `?subject=subject-1&subject=subject-2`'
    ),
  to: zod
    .date()
    .optional()
    .describe('End date-time in RFC 3339 format.\nInclusive.'),
})

/**
 * Get a notification event by id.
 * @summary Get notification event
 */
export const getNotificationEventParams = zod.object({
  eventId: zod.string(),
})

/**
 * List all notification rules.
 * @summary List notification rules
 */
export const listNotificationRulesQueryIncludeDeletedDefault = false
export const listNotificationRulesQueryIncludeDisabledDefault = false
export const listNotificationRulesQueryFeatureItemMax = 64

export const listNotificationRulesQueryFeatureItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listNotificationRulesQueryPageDefault = 1
export const listNotificationRulesQueryPageSizeDefault = 100
export const listNotificationRulesQueryPageSizeMax = 1000

export const listNotificationRulesQueryParams = zod.object({
  channel: zod
    .array(zod.string())
    .optional()
    .describe(
      'Filtering by multiple notifiaction channel ids.\n\nUsage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3`'
    ),
  feature: zod
    .array(
      zod
        .string()
        .min(1)
        .max(listNotificationRulesQueryFeatureItemMax)
        .regex(listNotificationRulesQueryFeatureItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).\nA key is a unique string that is used to identify a resource.\n\nTODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.'
        )
    )
    .optional()
    .describe(
      'Filtering by multiple feature ids/keys.\n\nUsage: `?feature=feature-1&feature=feature-2`'
    ),
  includeDeleted: zod
    .boolean()
    .optional()
    .describe(
      'Include deleted notification rules in response.\n\nUsage: `?includeDeleted=true`'
    ),
  includeDisabled: zod
    .boolean()
    .optional()
    .describe(
      'Include disabled notification rules in response.\n\nUsage: `?includeDisabled=false`'
    ),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'type', 'createdAt', 'updatedAt'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listNotificationRulesQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listNotificationRulesQueryPageSizeMax)
    .default(listNotificationRulesQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Create a new notification rule.
 * @summary Create a notification rule
 */
export const createNotificationRuleBodyDisabledDefault = false
export const createNotificationRuleBodyThresholdsMax = 10
export const createNotificationRuleBodyChannelsItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createNotificationRuleBodyFeaturesItemMax = 64

export const createNotificationRuleBodyFeaturesItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createNotificationRuleBodyDisabledDefaultOne = false
export const createNotificationRuleBodyChannelsItemRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createNotificationRuleBodyFeaturesItemMaxOne = 64

export const createNotificationRuleBodyFeaturesItemRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createNotificationRuleBodyDisabledDefaultTwo = false
export const createNotificationRuleBodyChannelsItemRegExpTwo = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createNotificationRuleBodyDisabledDefaultThree = false
export const createNotificationRuleBodyChannelsItemRegExpThree = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createNotificationRuleBody = zod
  .discriminatedUnion('type', [
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(createNotificationRuleBodyChannelsItemRegExp)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        features: zod
          .array(
            zod
              .string()
              .min(1)
              .max(createNotificationRuleBodyFeaturesItemMax)
              .regex(createNotificationRuleBodyFeaturesItemRegExp)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).\nA key is a unique string that is used to identify a resource.\n\nTODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.'
              )
          )
          .min(1)
          .optional()
          .describe(
            'Optional field for defining the scope of notification by feature. It may contain features by id or key.'
          ),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        thresholds: zod
          .array(
            zod
              .object({
                type: zod
                  .enum(['PERCENT', 'NUMBER'])
                  .describe(
                    'Type of the rule in the balance threshold specification.'
                  )
                  .describe('Type of the threshold.'),
                value: zod.number().describe('Value of the threshold.'),
              })
              .describe('Threshold value with multiple supported types.')
          )
          .min(1)
          .max(createNotificationRuleBodyThresholdsMax)
          .describe('List of thresholds the rule suppose to be triggered.'),
        type: zod.enum(['entitlements.balance.threshold']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with entitlements.balance.threshold type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(createNotificationRuleBodyChannelsItemRegExpOne)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        features: zod
          .array(
            zod
              .string()
              .min(1)
              .max(createNotificationRuleBodyFeaturesItemMaxOne)
              .regex(createNotificationRuleBodyFeaturesItemRegExpOne)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).\nA key is a unique string that is used to identify a resource.\n\nTODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.'
              )
          )
          .min(1)
          .optional()
          .describe(
            'Optional field for defining the scope of notification by feature. It may contain features by id or key.'
          ),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['entitlements.reset']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with entitlements.reset type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(createNotificationRuleBodyChannelsItemRegExpTwo)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['invoice.created']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with invoice.created type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(createNotificationRuleBodyChannelsItemRegExpThree)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['invoice.updated']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with invoice.updated  type.'
      ),
  ])
  .describe(
    'Union type for requests creating new notification rule with certain type.'
  )

/**
 * Update notification rule.
 * @summary Update a notification rule
 */
export const updateNotificationRulePathRuleIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateNotificationRuleParams = zod.object({
  ruleId: zod.string().regex(updateNotificationRulePathRuleIdRegExp),
})

export const updateNotificationRuleBodyDisabledDefault = false
export const updateNotificationRuleBodyThresholdsMax = 10
export const updateNotificationRuleBodyChannelsItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateNotificationRuleBodyFeaturesItemMax = 64

export const updateNotificationRuleBodyFeaturesItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateNotificationRuleBodyDisabledDefaultOne = false
export const updateNotificationRuleBodyChannelsItemRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateNotificationRuleBodyFeaturesItemMaxOne = 64

export const updateNotificationRuleBodyFeaturesItemRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateNotificationRuleBodyDisabledDefaultTwo = false
export const updateNotificationRuleBodyChannelsItemRegExpTwo = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateNotificationRuleBodyDisabledDefaultThree = false
export const updateNotificationRuleBodyChannelsItemRegExpThree = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateNotificationRuleBody = zod
  .discriminatedUnion('type', [
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(updateNotificationRuleBodyChannelsItemRegExp)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        features: zod
          .array(
            zod
              .string()
              .min(1)
              .max(updateNotificationRuleBodyFeaturesItemMax)
              .regex(updateNotificationRuleBodyFeaturesItemRegExp)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).\nA key is a unique string that is used to identify a resource.\n\nTODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.'
              )
          )
          .min(1)
          .optional()
          .describe(
            'Optional field for defining the scope of notification by feature. It may contain features by id or key.'
          ),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        thresholds: zod
          .array(
            zod
              .object({
                type: zod
                  .enum(['PERCENT', 'NUMBER'])
                  .describe(
                    'Type of the rule in the balance threshold specification.'
                  )
                  .describe('Type of the threshold.'),
                value: zod.number().describe('Value of the threshold.'),
              })
              .describe('Threshold value with multiple supported types.')
          )
          .min(1)
          .max(updateNotificationRuleBodyThresholdsMax)
          .describe('List of thresholds the rule suppose to be triggered.'),
        type: zod.enum(['entitlements.balance.threshold']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with entitlements.balance.threshold type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(updateNotificationRuleBodyChannelsItemRegExpOne)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        features: zod
          .array(
            zod
              .string()
              .min(1)
              .max(updateNotificationRuleBodyFeaturesItemMaxOne)
              .regex(updateNotificationRuleBodyFeaturesItemRegExpOne)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).\nA key is a unique string that is used to identify a resource.\n\nTODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.'
              )
          )
          .min(1)
          .optional()
          .describe(
            'Optional field for defining the scope of notification by feature. It may contain features by id or key.'
          ),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['entitlements.reset']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with entitlements.reset type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(updateNotificationRuleBodyChannelsItemRegExpTwo)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['invoice.created']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with invoice.created type.'
      ),
    zod
      .object({
        channels: zod
          .array(
            zod
              .string()
              .regex(updateNotificationRuleBodyChannelsItemRegExpThree)
              .describe(
                'ULID (Universally Unique Lexicographically Sortable Identifier).'
              )
          )
          .min(1)
          .describe('List of notification channels the rule is applied to.'),
        disabled: zod
          .boolean()
          .optional()
          .describe('Whether the rule is disabled or not.'),
        name: zod
          .string()
          .describe('The user friendly name of the notification rule.'),
        type: zod.enum(['invoice.updated']),
      })
      .describe(
        'Request with input parameters for creating new notification rule with invoice.updated  type.'
      ),
  ])
  .describe(
    'Union type for requests creating new notification rule with certain type.'
  )

/**
 * Get a notification rule by id.
 * @summary Get notification rule
 */
export const getNotificationRulePathRuleIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getNotificationRuleParams = zod.object({
  ruleId: zod.string().regex(getNotificationRulePathRuleIdRegExp),
})

/**
 * Soft delete notification rule by id.

Once a notification rule is deleted it cannot be undeleted.
 * @summary Delete a notification rule
 */
export const deleteNotificationRulePathRuleIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteNotificationRuleParams = zod.object({
  ruleId: zod.string().regex(deleteNotificationRulePathRuleIdRegExp),
})

/**
 * Test a notification rule by sending a test event with random data.
 * @summary Test notification rule
 */
export const testNotificationRulePathRuleIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const testNotificationRuleParams = zod.object({
  ruleId: zod.string().regex(testNotificationRulePathRuleIdRegExp),
})

/**
 * List all plans.
 * @summary List plans
 */
export const listPlansQueryIncludeDeletedDefault = false
export const listPlansQueryIdItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listPlansQueryKeyItemMax = 64

export const listPlansQueryKeyItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const listPlansQueryCurrencyItemMin = 3

export const listPlansQueryCurrencyItemMax = 3

export const listPlansQueryCurrencyItemRegExp = new RegExp('^[A-Z]{3}$')
export const listPlansQueryPageDefault = 1
export const listPlansQueryPageSizeDefault = 100
export const listPlansQueryPageSizeMax = 1000

export const listPlansQueryParams = zod.object({
  currency: zod
    .array(
      zod
        .string()
        .min(listPlansQueryCurrencyItemMin)
        .max(listPlansQueryCurrencyItemMax)
        .regex(listPlansQueryCurrencyItemRegExp)
        .describe(
          'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
        )
    )
    .optional()
    .describe('Filter by plan.currency attribute'),
  id: zod
    .array(
      zod
        .string()
        .regex(listPlansQueryIdItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by plan.id attribute'),
  includeDeleted: zod
    .boolean()
    .optional()
    .describe(
      'Include deleted plans in response.\n\nUsage: `?includeDeleted=true`'
    ),
  key: zod
    .array(
      zod
        .string()
        .min(1)
        .max(listPlansQueryKeyItemMax)
        .regex(listPlansQueryKeyItemRegExp)
        .describe(
          'A key is a unique string that is used to identify a resource.'
        )
    )
    .optional()
    .describe('Filter by plan.key attribute'),
  keyVersion: zod
    .record(zod.string(), zod.array(zod.number()))
    .optional()
    .describe('Filter by plan.key and plan.version attributes'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'key', 'version', 'created_at', 'updated_at'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listPlansQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listPlansQueryPageSizeMax)
    .default(listPlansQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
  status: zod
    .array(
      zod
        .enum(['draft', 'active', 'archived', 'scheduled'])
        .describe('The status of a plan.')
    )
    .optional()
    .describe(
      'Only return plans with the given status.\n\nUsage:\n- `?status=active`: return only the currently active plan\n- `?status=draft`: return only the draft plan\n- `?status=archived`: return only the archived plans'
    ),
})

/**
 * Create a new plan.
 * @summary Create a plan
 */
export const createPlanBodyNameMax = 256
export const createPlanBodyDescriptionMax = 1024
export const createPlanBodyKeyMax = 64

export const createPlanBodyKeyRegExp = new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createPlanBodyCurrencyMinOne = 3

export const createPlanBodyCurrencyMaxOne = 3

export const createPlanBodyCurrencyRegExpOne = new RegExp('^[A-Z]{3}$')
export const createPlanBodyCurrencyDefault = 'USD'
export const createPlanBodyProRatingConfigEnabledDefault = true
export const createPlanBodyProRatingConfigModeDefault = 'prorate_prices'
export const createPlanBodyProRatingConfigDefault = {
  enabled: true,
  mode: 'prorate_prices',
}
export const createPlanBodyPhasesItemKeyMax = 64

export const createPlanBodyPhasesItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createPlanBodyPhasesItemNameMax = 256
export const createPlanBodyPhasesItemDescriptionMax = 1024
export const createPlanBodyPhasesItemRateCardsItemKeyMax = 64

export const createPlanBodyPhasesItemRateCardsItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createPlanBodyPhasesItemRateCardsItemNameMax = 256
export const createPlanBodyPhasesItemRateCardsItemDescriptionMax = 1024
export const createPlanBodyPhasesItemRateCardsItemFeatureKeyMax = 64

export const createPlanBodyPhasesItemRateCardsItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const createPlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPricePaymentTermDefault =
  'in_advance'
export const createPlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemKeyMaxOne = 64

export const createPlanBodyPhasesItemRateCardsItemKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createPlanBodyPhasesItemRateCardsItemNameMaxOne = 256
export const createPlanBodyPhasesItemRateCardsItemDescriptionMaxOne = 1024
export const createPlanBodyPhasesItemRateCardsItemFeatureKeyMaxOne = 64

export const createPlanBodyPhasesItemRateCardsItemFeatureKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const createPlanBodyPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const createPlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMultiplierDefault = '1'
export const createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createPlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')

export const createPlanBody = zod
  .object({
    alignment: zod
      .object({
        billablesMustAlign: zod
          .boolean()
          .optional()
          .describe(
            "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
          ),
      })
      .describe('Alignment configuration for a plan or subscription.')
      .optional()
      .describe('Alignment configuration for the plan.'),
    billingCadence: zod
      .string()
      .describe(
        'The default billing cadence for subscriptions using this plan.\nDefines how often customers are billed using ISO8601 duration format.\nExamples: \"P1M\" (monthly), \"P3M\" (quarterly), \"P1Y\" (annually).'
      ),
    currency: zod
      .string()
      .min(createPlanBodyCurrencyMinOne)
      .max(createPlanBodyCurrencyMaxOne)
      .regex(createPlanBodyCurrencyRegExpOne)
      .describe(
        'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
      )
      .describe('The currency code of the plan.'),
    description: zod
      .string()
      .max(createPlanBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    key: zod
      .string()
      .min(1)
      .max(createPlanBodyKeyMax)
      .regex(createPlanBodyKeyRegExp)
      .describe('A semi-unique identifier for the resource.'),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createPlanBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    phases: zod
      .array(
        zod
          .object({
            description: zod
              .string()
              .max(createPlanBodyPhasesItemDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            duration: zod
              .string()
              .nullable()
              .describe('The duration of the phase.'),
            key: zod
              .string()
              .min(1)
              .max(createPlanBodyPhasesItemKeyMax)
              .regex(createPlanBodyPhasesItemKeyRegExp)
              .describe('A semi-unique identifier for the resource.'),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(createPlanBodyPhasesItemNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            rateCards: zod
              .array(
                zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .nullable()
                          .describe(
                            'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                          ),
                        description: zod
                          .string()
                          .max(
                            createPlanBodyPhasesItemRateCardsItemDescriptionMax
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                                  )
                                  .default(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            createPlanBodyPhasesItemRateCardsItemFeatureKeyMax
                          )
                          .regex(
                            createPlanBodyPhasesItemRateCardsItemFeatureKeyRegExp
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(createPlanBodyPhasesItemRateCardsItemKeyMax)
                          .regex(createPlanBodyPhasesItemRateCardsItemKeyRegExp)
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(createPlanBodyPhasesItemRateCardsItemNameMax)
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .object({
                            amount: zod
                              .string()
                              .regex(
                                createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .describe('The amount of the flat price.'),
                            paymentTerm: zod
                              .enum(['in_advance', 'in_arrears'])
                              .describe(
                                'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                              )
                              .default(
                                createPlanBodyPhasesItemRateCardsItemPricePaymentTermDefault
                              )
                              .describe(
                                'The payment term of the flat price.\nDefaults to in advance.'
                              ),
                            type: zod.enum(['flat']),
                          })
                          .describe('Flat price with payment term.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExp
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['flat_fee']),
                      })
                      .describe(
                        'A flat fee rate card defines a one-time purchase or a recurring fee.'
                      ),
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .describe('The billing cadence of the rate card.'),
                        description: zod
                          .string()
                          .max(
                            createPlanBodyPhasesItemRateCardsItemDescriptionMaxOne
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                                  )
                                  .default(
                                    createPlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            createPlanBodyPhasesItemRateCardsItemFeatureKeyMaxOne
                          )
                          .regex(
                            createPlanBodyPhasesItemRateCardsItemFeatureKeyRegExpOne
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(createPlanBodyPhasesItemRateCardsItemKeyMaxOne)
                          .regex(
                            createPlanBodyPhasesItemRateCardsItemKeyRegExpOne
                          )
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(createPlanBodyPhasesItemRateCardsItemNameMaxOne)
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                paymentTerm: zod
                                  .enum(['in_advance', 'in_arrears'])
                                  .describe(
                                    'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                  )
                                  .default(
                                    createPlanBodyPhasesItemRateCardsItemPricePaymentTermDefaultTwo
                                  )
                                  .describe(
                                    'The payment term of the flat price.\nDefaults to in advance.'
                                  ),
                                type: zod.enum(['flat']),
                              })
                              .describe('Flat price with payment term.'),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                type: zod.enum(['unit']),
                              })
                              .describe('Unit price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                mode: zod
                                  .enum(['volume', 'graduated'])
                                  .describe('The mode of the tiered price.')
                                  .describe(
                                    'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                                  ),
                                tiers: zod
                                  .array(
                                    zod
                                      .object({
                                        flatPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                createPlanBodyPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the flat price.'
                                              ),
                                            type: zod
                                              .enum(['flat'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Flat price.')
                                          .nullable()
                                          .describe(
                                            'The flat price component of the tier.'
                                          ),
                                        unitPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                createPlanBodyPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the unit price.'
                                              ),
                                            type: zod
                                              .enum(['unit'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Unit price.')
                                          .nullable()
                                          .describe(
                                            'The unit price component of the tier.'
                                          ),
                                        upToAmount: zod
                                          .string()
                                          .regex(
                                            createPlanBodyPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                          ),
                                      })
                                      .describe(
                                        'A price tier.\nAt least one price component is required in each tier.'
                                      )
                                  )
                                  .min(1)
                                  .describe(
                                    'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                                  ),
                                type: zod.enum(['tiered']),
                              })
                              .describe('Tiered price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                multiplier: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMultiplierRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .default(
                                    createPlanBodyPhasesItemRateCardsItemPriceMultiplierDefault
                                  )
                                  .describe(
                                    'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                                  ),
                                type: zod.enum(['dynamic']),
                              })
                              .describe(
                                'Dynamic price with spend commitments.'
                              ),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The price of one package.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                quantityPerPackage: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The quantity per package.'),
                                type: zod.enum(['package']),
                              })
                              .describe(
                                'Package price with spend commitments.'
                              ),
                          ])
                          .describe('The price of the usage based rate card.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    createPlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['usage_based']),
                      })
                      .describe(
                        'A usage-based rate card defines a price based on usage.'
                      ),
                  ])
                  .describe(
                    'A rate card defines the pricing and entitlement of a feature or service.'
                  )
              )
              .describe('The rate cards of the plan.'),
          })
          .describe(
            "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses."
          )
      )
      .min(1)
      .describe(
        "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.\nA phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices."
      ),
    proRatingConfig: zod
      .object({
        enabled: zod
          .boolean()
          .describe('Whether pro-rating is enabled for this plan.'),
        mode: zod
          .enum(['prorate_prices'])
          .describe(
            'Pro-rating mode options for handling billing period changes.'
          )
          .describe('How to handle pro-rating for billing period changes.'),
      })
      .describe('Configuration for pro-rating behavior.')
      .default(createPlanBodyProRatingConfigDefault)
      .describe(
        'Default pro-rating configuration for subscriptions using this plan.'
      ),
  })
  .describe('Resource create operation model.')

/**
 * Create a new draft version from plan.
It returns error if there is already a plan in draft or planId does not reference the latest published version.
 * @deprecated
 * @summary New draft plan
 */
export const nextPlanPathPlanIdOrKeyMax = 64

export const nextPlanPathPlanIdOrKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const nextPlanParams = zod.object({
  planIdOrKey: zod
    .string()
    .min(1)
    .max(nextPlanPathPlanIdOrKeyMax)
    .regex(nextPlanPathPlanIdOrKeyRegExp),
})

/**
 * Update plan by id.
 * @summary Update a plan
 */
export const updatePlanPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updatePlanParams = zod.object({
  planId: zod.string().regex(updatePlanPathPlanIdRegExp),
})

export const updatePlanBodyNameMax = 256
export const updatePlanBodyDescriptionMax = 1024
export const updatePlanBodyProRatingConfigEnabledDefault = true
export const updatePlanBodyProRatingConfigModeDefault = 'prorate_prices'
export const updatePlanBodyProRatingConfigDefault = {
  enabled: true,
  mode: 'prorate_prices',
}
export const updatePlanBodyPhasesItemKeyMax = 64

export const updatePlanBodyPhasesItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updatePlanBodyPhasesItemNameMax = 256
export const updatePlanBodyPhasesItemDescriptionMax = 1024
export const updatePlanBodyPhasesItemRateCardsItemKeyMax = 64

export const updatePlanBodyPhasesItemRateCardsItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updatePlanBodyPhasesItemRateCardsItemNameMax = 256
export const updatePlanBodyPhasesItemRateCardsItemDescriptionMax = 1024
export const updatePlanBodyPhasesItemRateCardsItemFeatureKeyMax = 64

export const updatePlanBodyPhasesItemRateCardsItemFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const updatePlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPricePaymentTermDefault =
  'in_advance'
export const updatePlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemKeyMaxOne = 64

export const updatePlanBodyPhasesItemRateCardsItemKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const updatePlanBodyPhasesItemRateCardsItemNameMaxOne = 256
export const updatePlanBodyPhasesItemRateCardsItemDescriptionMaxOne = 1024
export const updatePlanBodyPhasesItemRateCardsItemFeatureKeyMaxOne = 64

export const updatePlanBodyPhasesItemRateCardsItemFeatureKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const updatePlanBodyPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const updatePlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMultiplierDefault = '1'
export const updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const updatePlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')

export const updatePlanBody = zod
  .object({
    alignment: zod
      .object({
        billablesMustAlign: zod
          .boolean()
          .optional()
          .describe(
            "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
          ),
      })
      .describe('Alignment configuration for a plan or subscription.')
      .optional()
      .describe('Alignment configuration for the plan.'),
    billingCadence: zod
      .string()
      .describe(
        'The default billing cadence for subscriptions using this plan.\nDefines how often customers are billed using ISO8601 duration format.\nExamples: \"P1M\" (monthly), \"P3M\" (quarterly), \"P1Y\" (annually).'
      ),
    description: zod
      .string()
      .max(updatePlanBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updatePlanBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    phases: zod
      .array(
        zod
          .object({
            description: zod
              .string()
              .max(updatePlanBodyPhasesItemDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            duration: zod
              .string()
              .nullable()
              .describe('The duration of the phase.'),
            key: zod
              .string()
              .min(1)
              .max(updatePlanBodyPhasesItemKeyMax)
              .regex(updatePlanBodyPhasesItemKeyRegExp)
              .describe('A semi-unique identifier for the resource.'),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(updatePlanBodyPhasesItemNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            rateCards: zod
              .array(
                zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .nullable()
                          .describe(
                            'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                          ),
                        description: zod
                          .string()
                          .max(
                            updatePlanBodyPhasesItemRateCardsItemDescriptionMax
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                                  )
                                  .default(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            updatePlanBodyPhasesItemRateCardsItemFeatureKeyMax
                          )
                          .regex(
                            updatePlanBodyPhasesItemRateCardsItemFeatureKeyRegExp
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(updatePlanBodyPhasesItemRateCardsItemKeyMax)
                          .regex(updatePlanBodyPhasesItemRateCardsItemKeyRegExp)
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(updatePlanBodyPhasesItemRateCardsItemNameMax)
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .object({
                            amount: zod
                              .string()
                              .regex(
                                updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .describe('The amount of the flat price.'),
                            paymentTerm: zod
                              .enum(['in_advance', 'in_arrears'])
                              .describe(
                                'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                              )
                              .default(
                                updatePlanBodyPhasesItemRateCardsItemPricePaymentTermDefault
                              )
                              .describe(
                                'The payment term of the flat price.\nDefaults to in advance.'
                              ),
                            type: zod.enum(['flat']),
                          })
                          .describe('Flat price with payment term.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExp
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['flat_fee']),
                      })
                      .describe(
                        'A flat fee rate card defines a one-time purchase or a recurring fee.'
                      ),
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .describe('The billing cadence of the rate card.'),
                        description: zod
                          .string()
                          .max(
                            updatePlanBodyPhasesItemRateCardsItemDescriptionMaxOne
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                                  )
                                  .default(
                                    updatePlanBodyPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            updatePlanBodyPhasesItemRateCardsItemFeatureKeyMaxOne
                          )
                          .regex(
                            updatePlanBodyPhasesItemRateCardsItemFeatureKeyRegExpOne
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(updatePlanBodyPhasesItemRateCardsItemKeyMaxOne)
                          .regex(
                            updatePlanBodyPhasesItemRateCardsItemKeyRegExpOne
                          )
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(updatePlanBodyPhasesItemRateCardsItemNameMaxOne)
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                paymentTerm: zod
                                  .enum(['in_advance', 'in_arrears'])
                                  .describe(
                                    'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                  )
                                  .default(
                                    updatePlanBodyPhasesItemRateCardsItemPricePaymentTermDefaultTwo
                                  )
                                  .describe(
                                    'The payment term of the flat price.\nDefaults to in advance.'
                                  ),
                                type: zod.enum(['flat']),
                              })
                              .describe('Flat price with payment term.'),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                type: zod.enum(['unit']),
                              })
                              .describe('Unit price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                mode: zod
                                  .enum(['volume', 'graduated'])
                                  .describe('The mode of the tiered price.')
                                  .describe(
                                    'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                                  ),
                                tiers: zod
                                  .array(
                                    zod
                                      .object({
                                        flatPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                updatePlanBodyPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the flat price.'
                                              ),
                                            type: zod
                                              .enum(['flat'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Flat price.')
                                          .nullable()
                                          .describe(
                                            'The flat price component of the tier.'
                                          ),
                                        unitPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                updatePlanBodyPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the unit price.'
                                              ),
                                            type: zod
                                              .enum(['unit'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Unit price.')
                                          .nullable()
                                          .describe(
                                            'The unit price component of the tier.'
                                          ),
                                        upToAmount: zod
                                          .string()
                                          .regex(
                                            updatePlanBodyPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                          ),
                                      })
                                      .describe(
                                        'A price tier.\nAt least one price component is required in each tier.'
                                      )
                                  )
                                  .min(1)
                                  .describe(
                                    'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                                  ),
                                type: zod.enum(['tiered']),
                              })
                              .describe('Tiered price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                multiplier: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMultiplierRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .default(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMultiplierDefault
                                  )
                                  .describe(
                                    'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                                  ),
                                type: zod.enum(['dynamic']),
                              })
                              .describe(
                                'Dynamic price with spend commitments.'
                              ),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The price of one package.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                quantityPerPackage: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The quantity per package.'),
                                type: zod.enum(['package']),
                              })
                              .describe(
                                'Package price with spend commitments.'
                              ),
                          ])
                          .describe('The price of the usage based rate card.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    updatePlanBodyPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['usage_based']),
                      })
                      .describe(
                        'A usage-based rate card defines a price based on usage.'
                      ),
                  ])
                  .describe(
                    'A rate card defines the pricing and entitlement of a feature or service.'
                  )
              )
              .describe('The rate cards of the plan.'),
          })
          .describe(
            "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses."
          )
      )
      .min(1)
      .describe(
        "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.\nA phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices."
      ),
    proRatingConfig: zod
      .object({
        enabled: zod
          .boolean()
          .describe('Whether pro-rating is enabled for this plan.'),
        mode: zod
          .enum(['prorate_prices'])
          .describe(
            'Pro-rating mode options for handling billing period changes.'
          )
          .describe('How to handle pro-rating for billing period changes.'),
      })
      .describe('Configuration for pro-rating behavior.')
      .default(updatePlanBodyProRatingConfigDefault)
      .describe(
        'Default pro-rating configuration for subscriptions using this plan.'
      ),
  })
  .describe('Resource update operation model.')

/**
 * Get a plan by id or key. The latest published version is returned if latter is used.
 * @summary Get plan
 */
export const getPlanPathPlanIdMax = 64

export const getPlanPathPlanIdRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getPlanParams = zod.object({
  planId: zod
    .string()
    .min(1)
    .max(getPlanPathPlanIdMax)
    .regex(getPlanPathPlanIdRegExp),
})

export const getPlanQueryIncludeLatestDefault = false

export const getPlanQueryParams = zod.object({
  includeLatest: zod
    .boolean()
    .optional()
    .describe(
      'Include latest version of the Plan instead of the version in active state.\n\nUsage: `?includeLatest=true`'
    ),
})

/**
 * Soft delete plan by plan.id.

Once a plan is deleted it cannot be undeleted.
 * @summary Delete plan
 */
export const deletePlanPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deletePlanParams = zod.object({
  planId: zod.string().regex(deletePlanPathPlanIdRegExp),
})

/**
 * List all available add-ons for plan.
 * @summary List all available add-ons for plan
 */
export const listPlanAddonsPathPlanIdMax = 64

export const listPlanAddonsPathPlanIdRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const listPlanAddonsParams = zod.object({
  planId: zod
    .string()
    .min(1)
    .max(listPlanAddonsPathPlanIdMax)
    .regex(listPlanAddonsPathPlanIdRegExp),
})

export const listPlanAddonsQueryIncludeDeletedDefault = false
export const listPlanAddonsQueryIdItemRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const listPlanAddonsQueryKeyItemMax = 64

export const listPlanAddonsQueryKeyItemRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const listPlanAddonsQueryPageDefault = 1
export const listPlanAddonsQueryPageSizeDefault = 100
export const listPlanAddonsQueryPageSizeMax = 1000

export const listPlanAddonsQueryParams = zod.object({
  id: zod
    .array(
      zod
        .string()
        .regex(listPlanAddonsQueryIdItemRegExp)
        .describe(
          'ULID (Universally Unique Lexicographically Sortable Identifier).'
        )
    )
    .optional()
    .describe('Filter by addon.id attribute.'),
  includeDeleted: zod
    .boolean()
    .optional()
    .describe(
      'Include deleted plan add-on assignments.\n\nUsage: `?includeDeleted=true`'
    ),
  key: zod
    .array(
      zod
        .string()
        .min(1)
        .max(listPlanAddonsQueryKeyItemMax)
        .regex(listPlanAddonsQueryKeyItemRegExp)
        .describe(
          'A key is a unique string that is used to identify a resource.'
        )
    )
    .optional()
    .describe('Filter by addon.key attribute.'),
  keyVersion: zod
    .record(zod.string(), zod.array(zod.number()))
    .optional()
    .describe('Filter by addon.key and addon.version attributes.'),
  order: zod.enum(['ASC', 'DESC']).optional().describe('The order direction.'),
  orderBy: zod
    .enum(['id', 'key', 'version', 'created_at', 'updated_at'])
    .optional()
    .describe('The order by field.'),
  page: zod
    .number()
    .min(1)
    .default(listPlanAddonsQueryPageDefault)
    .describe('Page index.\n\nDefault is 1.'),
  pageSize: zod
    .number()
    .min(1)
    .max(listPlanAddonsQueryPageSizeMax)
    .default(listPlanAddonsQueryPageSizeDefault)
    .describe('The maximum number of items per page.\n\nDefault is 100.'),
})

/**
 * Create new add-on assignment for plan.
 * @summary Create new add-on assignment for plan
 */
export const createPlanAddonPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createPlanAddonParams = zod.object({
  planId: zod.string().regex(createPlanAddonPathPlanIdRegExp),
})

export const createPlanAddonBodyAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createPlanAddonBody = zod
  .object({
    addonId: zod
      .string()
      .regex(createPlanAddonBodyAddonIdRegExp)
      .describe('The add-on unique identifier in ULID format.'),
    fromPlanPhase: zod
      .string()
      .describe(
        'The key of the plan phase from the add-on becomes available for purchase.'
      ),
    maxQuantity: zod
      .number()
      .optional()
      .describe(
        'The maximum number of times the add-on can be purchased for the plan.\nIt is not applicable for add-ons with single instance type.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional()
      .describe('Additional metadata for the resource.'),
  })
  .describe('A plan add-on assignment create request.')

/**
 * Update add-on assignment for plan.
 * @summary Update add-on assignment for plan
 */
export const updatePlanAddonPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updatePlanAddonPathPlanAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updatePlanAddonParams = zod.object({
  planAddonId: zod.string().regex(updatePlanAddonPathPlanAddonIdRegExp),
  planId: zod.string().regex(updatePlanAddonPathPlanIdRegExp),
})

export const updatePlanAddonBody = zod
  .object({
    fromPlanPhase: zod
      .string()
      .describe(
        'The key of the plan phase from the add-on becomes available for purchase.'
      ),
    maxQuantity: zod
      .number()
      .optional()
      .describe(
        'The maximum number of times the add-on can be purchased for the plan.\nIt is not applicable for add-ons with single instance type.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional()
      .describe('Additional metadata for the resource.'),
  })
  .describe('Resource update operation model.')

/**
 * Get add-on assignment for plan by id.
 * @summary Get add-on assignment for plan
 */
export const getPlanAddonPathPlanIdMax = 64

export const getPlanAddonPathPlanIdRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const getPlanAddonPathPlanAddonIdMax = 64

export const getPlanAddonPathPlanAddonIdRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getPlanAddonParams = zod.object({
  planAddonId: zod
    .string()
    .min(1)
    .max(getPlanAddonPathPlanAddonIdMax)
    .regex(getPlanAddonPathPlanAddonIdRegExp),
  planId: zod
    .string()
    .min(1)
    .max(getPlanAddonPathPlanIdMax)
    .regex(getPlanAddonPathPlanIdRegExp),
})

/**
 * Delete add-on assignment for plan.

Once a plan is deleted it cannot be undeleted.
 * @summary Delete add-on assignment for plan
 */
export const deletePlanAddonPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const deletePlanAddonPathPlanAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deletePlanAddonParams = zod.object({
  planAddonId: zod.string().regex(deletePlanAddonPathPlanAddonIdRegExp),
  planId: zod.string().regex(deletePlanAddonPathPlanIdRegExp),
})

/**
 * Archive a plan version.
 * @summary Archive plan version
 */
export const archivePlanPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const archivePlanParams = zod.object({
  planId: zod.string().regex(archivePlanPathPlanIdRegExp),
})

/**
 * Publish a plan version.
 * @summary Publish plan
 */
export const publishPlanPathPlanIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const publishPlanParams = zod.object({
  planId: zod.string().regex(publishPlanPathPlanIdRegExp),
})

/**
 * Query meter for consumer portal. This endpoint is publicly exposable to consumers. Query meter for consumer portal. This endpoint is publicly exposable to consumers.
 * @summary Query meter Query meter
 */
export const queryPortalMeterPathMeterSlugMax = 64

export const queryPortalMeterPathMeterSlugRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)

export const queryPortalMeterParams = zod.object({
  meterSlug: zod
    .string()
    .min(1)
    .max(queryPortalMeterPathMeterSlugMax)
    .regex(queryPortalMeterPathMeterSlugRegExp),
})

export const queryPortalMeterQueryClientIdMax = 36
export const queryPortalMeterQueryWindowTimeZoneDefault = 'UTC'

export const queryPortalMeterQueryParams = zod.object({
  clientId: zod
    .string()
    .min(1)
    .max(queryPortalMeterQueryClientIdMax)
    .optional()
    .describe('Client ID\nUseful to track progress of a query.'),
  filterGroupBy: zod
    .record(zod.string(), zod.string())
    .optional()
    .describe(
      'Simple filter for group bys with exact match.\n\nFor example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo'
    ),
  from: zod
    .date()
    .optional()
    .describe(
      'Start date-time in RFC 3339 format.\n\nInclusive.\n\nFor example: ?from=2025-01-01T00%3A00%3A00.000Z'
    ),
  groupBy: zod
    .array(zod.string())
    .optional()
    .describe(
      'If not specified a single aggregate will be returned for each subject and time window.\n`subject` is a reserved group by value.\n\nFor example: ?groupBy=subject&groupBy=model'
    ),
  to: zod
    .date()
    .optional()
    .describe(
      'End date-time in RFC 3339 format.\n\nInclusive.\n\nFor example: ?to=2025-02-01T00%3A00%3A00.000Z'
    ),
  windowSize: zod
    .enum(['MINUTE', 'HOUR', 'DAY'])
    .optional()
    .describe(
      'If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.\n\nFor example: ?windowSize=DAY'
    ),
  windowTimeZone: zod
    .string()
    .default(queryPortalMeterQueryWindowTimeZoneDefault)
    .describe(
      'The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).\nIf not specified, the UTC timezone will be used.\n\nFor example: ?windowTimeZone=UTC'
    ),
})

/**
 * Create a consumer portal token.
 * @summary Create consumer portal token
 */
export const createPortalTokenBody = zod
  .object({
    allowedMeterSlugs: zod
      .array(zod.string())
      .optional()
      .describe(
        'Optional, if defined only the specified meters will be allowed.'
      ),
    subject: zod.string(),
  })
  .describe(
    "A consumer portal token.\n\nValidator doesn't obey required for readOnly properties\nSee: https://github.com/stoplightio/spectral/issues/1274"
  )

/**
 * List tokens.
 * @summary List consumer portal tokens
 */
export const listPortalTokensQueryLimitDefault = 25
export const listPortalTokensQueryLimitMax = 100

export const listPortalTokensQueryParams = zod.object({
  limit: zod
    .number()
    .min(1)
    .max(listPortalTokensQueryLimitMax)
    .default(listPortalTokensQueryLimitDefault),
})

/**
 * Invalidates consumer portal tokens by ID or subject.
 * @summary Invalidate portal tokens
 */
export const invalidatePortalTokensBody = zod.object({
  id: zod.string().optional().describe('Invalidate a portal token by ID.'),
  subject: zod
    .string()
    .optional()
    .describe('Invalidate all portal tokens for a subject.'),
})

/**
 * Create checkout session.
 * @summary Create checkout session
 */
export const createStripeCheckoutSessionBodyAppIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createStripeCheckoutSessionBodyCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createStripeCheckoutSessionBodyCustomerNameMax = 256
export const createStripeCheckoutSessionBodyCustomerDescriptionMax = 1024
export const createStripeCheckoutSessionBodyCustomerKeyMaxOne = 256
export const createStripeCheckoutSessionBodyCustomerUsageAttributionSubjectKeysMax = 1
export const createStripeCheckoutSessionBodyCustomerCurrencyMinOne = 3

export const createStripeCheckoutSessionBodyCustomerCurrencyMaxOne = 3

export const createStripeCheckoutSessionBodyCustomerCurrencyRegExpOne =
  new RegExp('^[A-Z]{3}$')
export const createStripeCheckoutSessionBodyCustomerBillingAddressCountryMinOne = 2

export const createStripeCheckoutSessionBodyCustomerBillingAddressCountryMaxOne = 2

export const createStripeCheckoutSessionBodyCustomerBillingAddressCountryRegExpOne =
  new RegExp('^[A-Z]{2}$')
export const createStripeCheckoutSessionBodyOptionsCurrencyMinOne = 3

export const createStripeCheckoutSessionBodyOptionsCurrencyMaxOne = 3

export const createStripeCheckoutSessionBodyOptionsCurrencyRegExpOne =
  new RegExp('^[A-Z]{3}$')
export const createStripeCheckoutSessionBodyOptionsCustomTextAfterSubmitMessageMax = 1200
export const createStripeCheckoutSessionBodyOptionsCustomTextShippingAddressMessageMax = 1200
export const createStripeCheckoutSessionBodyOptionsCustomTextSubmitMessageMax = 1200
export const createStripeCheckoutSessionBodyOptionsCustomTextTermsOfServiceAcceptanceMessageMax = 1200

export const createStripeCheckoutSessionBody = zod
  .object({
    appId: zod
      .string()
      .regex(createStripeCheckoutSessionBodyAppIdRegExp)
      .optional()
      .describe('If not provided, the default Stripe app is used if any.'),
    customer: zod
      .object({
        id: zod
          .string()
          .regex(createStripeCheckoutSessionBodyCustomerIdRegExp)
          .describe(
            'ULID (Universally Unique Lexicographically Sortable Identifier).'
          ),
      })
      .describe('Create Stripe checkout session with customer ID.')
      .or(
        zod
          .object({
            key: zod.string(),
          })
          .describe('Create Stripe checkout session with customer key.')
      )
      .or(
        zod
          .object({
            billingAddress: zod
              .object({
                city: zod.string().optional().describe('City.'),
                country: zod
                  .string()
                  .min(
                    createStripeCheckoutSessionBodyCustomerBillingAddressCountryMinOne
                  )
                  .max(
                    createStripeCheckoutSessionBodyCustomerBillingAddressCountryMaxOne
                  )
                  .regex(
                    createStripeCheckoutSessionBodyCustomerBillingAddressCountryRegExpOne
                  )
                  .describe(
                    '[ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.\nCustom two-letter country codes are also supported for convenience.'
                  )
                  .optional()
                  .describe(
                    'Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.'
                  ),
                line1: zod
                  .string()
                  .optional()
                  .describe('First line of the address.'),
                line2: zod
                  .string()
                  .optional()
                  .describe('Second line of the address.'),
                phoneNumber: zod.string().optional().describe('Phone number.'),
                postalCode: zod.string().optional().describe('Postal code.'),
                state: zod.string().optional().describe('State or province.'),
              })
              .describe('Address')
              .optional()
              .describe(
                'The billing address of the customer.\nUsed for tax and invoicing.'
              ),
            currency: zod
              .string()
              .min(createStripeCheckoutSessionBodyCustomerCurrencyMinOne)
              .max(createStripeCheckoutSessionBodyCustomerCurrencyMaxOne)
              .regex(createStripeCheckoutSessionBodyCustomerCurrencyRegExpOne)
              .describe(
                'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
              )
              .optional()
              .describe(
                'Currency of the customer.\nUsed for billing, tax and invoicing.'
              ),
            description: zod
              .string()
              .max(createStripeCheckoutSessionBodyCustomerDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            key: zod
              .string()
              .min(1)
              .max(createStripeCheckoutSessionBodyCustomerKeyMaxOne)
              .optional()
              .describe(
                'An optional unique key of the customer.\nUseful to reference the customer in external systems.\nFor example, your database ID.'
              ),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(createStripeCheckoutSessionBodyCustomerNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            primaryEmail: zod
              .string()
              .optional()
              .describe('The primary email address of the customer.'),
            usageAttribution: zod
              .object({
                subjectKeys: zod
                  .array(zod.string())
                  .min(1)
                  .max(
                    createStripeCheckoutSessionBodyCustomerUsageAttributionSubjectKeysMax
                  )
                  .describe(
                    'The subjects that are attributed to the customer.'
                  ),
              })
              .describe(
                'Mapping to attribute metered usage to the customer.\nOne customer can have multiple subjects,\nbut one subject can only belong to one customer.'
              )
              .describe('Mapping to attribute metered usage to the customer'),
          })
          .describe('Resource create operation model.')
      )
      .describe(
        'Provide a customer ID or key to use an existing OpenMeter customer.\nor provide a customer object to create a new customer.'
      ),
    options: zod
      .object({
        billingAddressCollection: zod
          .enum(['auto', 'required'])
          .describe(
            'Specify whether Checkout should collect the customer’s billing address.'
          )
          .optional()
          .describe(
            'Specify whether Checkout should collect the customer’s billing address. Defaults to auto.'
          ),
        cancelURL: zod
          .string()
          .optional()
          .describe(
            'If set, Checkout displays a back button and customers will be directed to this URL if they decide to cancel payment and return to your website.\nThis parameter is not allowed if ui_mode is embedded.'
          ),
        clientReferenceID: zod
          .string()
          .optional()
          .describe(
            'A unique string to reference the Checkout Session. This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal systems.'
          ),
        consentCollection: zod
          .object({
            paymentMethodReuseAgreement: zod
              .object({
                position: zod
                  .enum(['auto', 'hidden'])
                  .optional()
                  .describe(
                    'Create Stripe checkout session consent collection agreement position.'
                  ),
              })
              .describe(
                'Create Stripe checkout session payment method reuse agreement.'
              )
              .optional()
              .describe(
                'Determines the position and visibility of the payment method reuse agreement in the UI.\nWhen set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse agreement text will always be hidden in the UI.'
              ),
            promotions: zod
              .enum(['auto', 'none'])
              .describe(
                'Create Stripe checkout session consent collection promotions.'
              )
              .optional()
              .describe(
                'If set to auto, enables the collection of customer consent for promotional communications.\nThe Checkout Session will determine whether to display an option to opt into promotional\ncommunication from the merchant depending on the customer’s locale. Only available to US merchants.'
              ),
            termsOfService: zod
              .enum(['none', 'required'])
              .describe(
                'Create Stripe checkout session consent collection terms of service.'
              )
              .optional()
              .describe(
                'If set to required, it requires customers to check a terms of service checkbox before being able to pay.\nThere must be a valid terms of service URL set in your Stripe Dashboard settings.\nhttps://dashboard.stripe.com/settings/public'
              ),
          })
          .describe(
            'Configure fields for the Checkout Session to gather active consent from customers.'
          )
          .optional()
          .describe(
            'Configure fields for the Checkout Session to gather active consent from customers.'
          ),
        currency: zod
          .string()
          .min(createStripeCheckoutSessionBodyOptionsCurrencyMinOne)
          .max(createStripeCheckoutSessionBodyOptionsCurrencyMaxOne)
          .regex(createStripeCheckoutSessionBodyOptionsCurrencyRegExpOne)
          .describe(
            'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
          )
          .optional()
          .describe('Three-letter ISO currency code, in lowercase.'),
        customerUpdate: zod
          .object({
            address: zod
              .enum(['auto', 'never'])
              .describe(
                'Create Stripe checkout session customer update behavior.'
              )
              .optional()
              .describe(
                'Describes whether Checkout saves the billing address onto customer.address.\nTo always collect a full billing address, use billing_address_collection.\nDefaults to never.'
              ),
            name: zod
              .enum(['auto', 'never'])
              .describe(
                'Create Stripe checkout session customer update behavior.'
              )
              .optional()
              .describe(
                'Describes whether Checkout saves the name onto customer.name.\nDefaults to never.'
              ),
            shipping: zod
              .enum(['auto', 'never'])
              .describe(
                'Create Stripe checkout session customer update behavior.'
              )
              .optional()
              .describe(
                'Describes whether Checkout saves shipping information onto customer.shipping.\nTo collect shipping information, use shipping_address_collection.\nDefaults to never.'
              ),
          })
          .describe(
            'Controls what fields on Customer can be updated by the Checkout Session.'
          )
          .optional()
          .describe(
            'Controls what fields on Customer can be updated by the Checkout Session.'
          ),
        customText: zod
          .object({
            afterSubmit: zod
              .object({
                message: zod
                  .string()
                  .max(
                    createStripeCheckoutSessionBodyOptionsCustomTextAfterSubmitMessageMax
                  )
                  .optional(),
              })
              .optional()
              .describe(
                'Custom text that should be displayed after the payment confirmation button.'
              ),
            shippingAddress: zod
              .object({
                message: zod
                  .string()
                  .max(
                    createStripeCheckoutSessionBodyOptionsCustomTextShippingAddressMessageMax
                  )
                  .optional(),
              })
              .optional()
              .describe(
                'Custom text that should be displayed alongside shipping address collection.'
              ),
            submit: zod
              .object({
                message: zod
                  .string()
                  .max(
                    createStripeCheckoutSessionBodyOptionsCustomTextSubmitMessageMax
                  )
                  .optional(),
              })
              .optional()
              .describe(
                'Custom text that should be displayed alongside the payment confirmation button.'
              ),
            termsOfServiceAcceptance: zod
              .object({
                message: zod
                  .string()
                  .max(
                    createStripeCheckoutSessionBodyOptionsCustomTextTermsOfServiceAcceptanceMessageMax
                  )
                  .optional(),
              })
              .optional()
              .describe(
                'Custom text that should be displayed in place of the default terms of service agreement text.'
              ),
          })
          .describe('Stripe CheckoutSession.custom_text')
          .optional()
          .describe(
            'Display additional text for your customers using custom text.'
          ),
        expiresAt: zod
          .number()
          .optional()
          .describe(
            'The Epoch time in seconds at which the Checkout Session will expire.\nIt can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default, this value is 24 hours from creation.'
          ),
        locale: zod.string().optional(),
        metadata: zod
          .record(zod.string(), zod.string())
          .optional()
          .describe(
            'Set of key-value pairs that you can attach to an object.\nThis can be useful for storing additional information about the object in a structured format.\nIndividual keys can be unset by posting an empty value to them.\nAll keys can be unset by posting an empty value to metadata.'
          ),
        paymentMethodTypes: zod
          .array(zod.string())
          .optional()
          .describe(
            'A list of the types of payment methods (e.g., card) this Checkout Session can accept.'
          ),
        redirectOnCompletion: zod
          .enum(['always', 'if_required', 'never'])
          .describe('Create Stripe checkout session redirect on completion.')
          .optional()
          .describe(
            'This parameter applies to ui_mode: embedded. Defaults to always.\nLearn more about the redirect behavior of embedded sessions at\nhttps://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form'
          ),
        returnURL: zod
          .string()
          .optional()
          .describe(
            'The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method’s app or site.\nThis parameter is required if ui_mode is embedded and redirect-based payment methods are enabled on the session.'
          ),
        successURL: zod
          .string()
          .optional()
          .describe(
            'The URL to which Stripe should send customers when payment or setup is complete.\nThis parameter is not allowed if ui_mode is embedded.\nIf you’d like to use information from the successful Checkout Session on your page, read the guide on customizing your success page:\nhttps://docs.stripe.com/payments/checkout/custom-success-page'
          ),
        taxIdCollection: zod
          .object({
            enabled: zod
              .boolean()
              .describe(
                'Enable tax ID collection during checkout. Defaults to false.'
              ),
            required: zod
              .enum(['if_supported', 'never'])
              .describe(
                'Create Stripe checkout session tax ID collection required.'
              )
              .optional()
              .describe(
                'Describes whether a tax ID is required during checkout. Defaults to never.'
              ),
          })
          .describe('Create Stripe checkout session tax ID collection.')
          .optional()
          .describe('Controls tax ID collection during checkout.'),
        uiMode: zod
          .enum(['embedded', 'hosted'])
          .describe('Stripe CheckoutSession.ui_mode')
          .optional()
          .describe('The UI mode of the Session. Defaults to hosted.'),
      })
      .describe(
        'Create Stripe checkout session options\nSee https://docs.stripe.com/api/checkout/sessions/create'
      )
      .describe('Options passed to Stripe when creating the checkout session.'),
    stripeCustomerId: zod
      .string()
      .optional()
      .describe(
        "Stripe customer ID.\nIf not provided OpenMeter creates a new Stripe customer or\nuses the OpenMeter customer's default Stripe customer ID."
      ),
  })
  .describe('Create Stripe checkout session request.')

/**
 * Upserts a subject. Creates or updates subject.

If the subject doesn't exist, it will be created.
If the subject exists, it will be partially updated with the provided fields.
 * @summary Upsert subject
 */
export const upsertSubjectBodyItem = zod
  .object({
    currentPeriodEnd: zod
      .date()
      .optional()
      .describe('The end of the current period for the subject.'),
    currentPeriodStart: zod
      .date()
      .optional()
      .describe('The start of the current period for the subject.'),
    displayName: zod
      .string()
      .nullish()
      .describe('A human-readable display name for the subject.'),
    key: zod
      .string()
      .describe(
        'A unique, human-readable identifier for the subject.\nThis is typically a database ID or a customer key.'
      ),
    metadata: zod
      .record(zod.string(), zod.any())
      .nullish()
      .describe('Metadata for the subject.'),
    stripeCustomerId: zod
      .string()
      .nullish()
      .describe('The Stripe customer ID for the subject.'),
  })
  .describe('A subject is a unique identifier for a user or entity.')
export const upsertSubjectBody = zod.array(upsertSubjectBodyItem)

/**
 * Get subject by ID or key.
 * @summary Get subject
 */
export const getSubjectParams = zod.object({
  subjectIdOrKey: zod.string(),
})

/**
 * Delete subject by ID or key.
 * @summary Delete subject
 */
export const deleteSubjectParams = zod.object({
  subjectIdOrKey: zod.string(),
})

/**
 * OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).

- Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
- Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
- Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.

A given subject can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.

Once an entitlement is created you cannot modify it, only delete it.
 * @summary Create an entitlement
 */
export const createEntitlementParams = zod.object({
  subjectIdOrKey: zod.string(),
})

export const createEntitlementBodyFeatureKeyMax = 64

export const createEntitlementBodyFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createEntitlementBodyFeatureIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createEntitlementBodyIsSoftLimitDefault = false
export const createEntitlementBodyIsUnlimitedDefault = false
export const createEntitlementBodyIssueAfterResetMin = 0
export const createEntitlementBodyIssueAfterResetPriorityDefault = 1
export const createEntitlementBodyIssueAfterResetPriorityMax = 255
export const createEntitlementBodyPreserveOverageAtResetDefault = false
export const createEntitlementBodyFeatureKeyMaxOne = 64

export const createEntitlementBodyFeatureKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createEntitlementBodyFeatureIdRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createEntitlementBodyFeatureKeyMaxTwo = 64

export const createEntitlementBodyFeatureKeyRegExpTwo = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createEntitlementBodyFeatureIdRegExpTwo = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createEntitlementBody = zod
  .discriminatedUnion('type', [
    zod
      .object({
        featureId: zod
          .string()
          .regex(createEntitlementBodyFeatureIdRegExp)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(createEntitlementBodyFeatureKeyMax)
          .regex(createEntitlementBodyFeatureKeyRegExp)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        isSoftLimit: zod
          .boolean()
          .optional()
          .describe(
            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
          ),
        issueAfterReset: zod
          .number()
          .min(createEntitlementBodyIssueAfterResetMin)
          .optional()
          .describe(
            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
          ),
        issueAfterResetPriority: zod
          .number()
          .min(1)
          .max(createEntitlementBodyIssueAfterResetPriorityMax)
          .default(createEntitlementBodyIssueAfterResetPriorityDefault)
          .describe('Defines the grant priority for the default grant.'),
        isUnlimited: zod
          .boolean()
          .optional()
          .describe(
            'Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed in the future.'
          ),
        measureUsageFrom: zod
          .enum(['CURRENT_PERIOD_START', 'NOW'])
          .describe('Start of measurement options')
          .or(
            zod
              .date()
              .describe(
                '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
              )
          )
          .describe('Measure usage from')
          .optional()
          .describe(
            'Defines the time from which usage is measured. If not specified on creation, defaults to entitlement creation time.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        preserveOverageAtReset: zod
          .boolean()
          .optional()
          .describe(
            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
          ),
        type: zod.enum(['metered']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inpurs for metered entitlement'),
    zod
      .object({
        config: zod
          .string()
          .describe(
            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
          ),
        featureId: zod
          .string()
          .regex(createEntitlementBodyFeatureIdRegExpOne)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(createEntitlementBodyFeatureKeyMaxOne)
          .regex(createEntitlementBodyFeatureKeyRegExpOne)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        type: zod.enum(['static']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .optional()
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inputs for static entitlement'),
    zod
      .object({
        featureId: zod
          .string()
          .regex(createEntitlementBodyFeatureIdRegExpTwo)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(createEntitlementBodyFeatureKeyMaxTwo)
          .regex(createEntitlementBodyFeatureKeyRegExpTwo)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        type: zod.enum(['boolean']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .optional()
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inputs for boolean entitlement'),
  ])
  .describe('Create inputs for entitlement')

/**
 * List all entitlements for a subject. For checking entitlement access, use the /value endpoint instead.
 * @summary List entitlements
 */
export const listSubjectEntitlementsParams = zod.object({
  subjectIdOrKey: zod.string(),
})

export const listSubjectEntitlementsQueryIncludeDeletedDefault = false

export const listSubjectEntitlementsQueryParams = zod.object({
  includeDeleted: zod.boolean().optional(),
})

/**
 * List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
 * @summary List entitlement grants
 */
export const listEntitlementGrantsParams = zod.object({
  entitlementIdOrFeatureKey: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const listEntitlementGrantsQueryIncludeDeletedDefault = false

export const listEntitlementGrantsQueryParams = zod.object({
  includeDeleted: zod.boolean().optional(),
  orderBy: zod.enum(['id', 'createdAt', 'updatedAt']).optional(),
})

/**
 * Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.

A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.

Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.

Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).

Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))

Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.
 * @summary Create grant
 */
export const createGrantParams = zod.object({
  entitlementIdOrFeatureKey: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const createGrantBodyAmountMin = 0
export const createGrantBodyPriorityMax = 255
export const createGrantBodyMaxRolloverAmountDefault = 0
export const createGrantBodyMinRolloverAmountDefault = 0

export const createGrantBody = zod
  .object({
    amount: zod
      .number()
      .min(createGrantBodyAmountMin)
      .describe('The amount to grant. Should be a positive number.'),
    effectiveAt: zod
      .date()
      .describe(
        'Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute).'
      ),
    expiration: zod
      .object({
        count: zod
          .number()
          .describe('The number of time units in the expiration period.'),
        duration: zod
          .enum(['HOUR', 'DAY', 'WEEK', 'MONTH', 'YEAR'])
          .describe('The expiration duration enum')
          .describe('The unit of time for the expiration period.'),
      })
      .describe('The grant expiration definition')
      .describe('The grant expiration definition'),
    maxRolloverAmount: zod
      .number()
      .optional()
      .describe(
        'Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.\nBalance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional()
      .describe('The grant metadata.'),
    minRolloverAmount: zod
      .number()
      .optional()
      .describe(
        'Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.\nBalance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))'
      ),
    priority: zod
      .number()
      .min(1)
      .max(createGrantBodyPriorityMax)
      .optional()
      .describe(
        'The priority of the grant. Grants with higher priority are applied first.\nPriority is a positive decimal numbers. With lower numbers indicating higher importance.\nFor example, a priority of 1 is more urgent than a priority of 2.\nWhen there are several grants available for the same subject, the system selects the grant with the highest priority.\nIn cases where grants share the same priority level, the grant closest to its expiration will be used first.\nIn the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first.'
      ),
    recurrence: zod
      .object({
        anchor: zod
          .date()
          .optional()
          .describe('A date-time anchor to base the recurring period on.'),
        interval: zod
          .string()
          .or(
            zod
              .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
              .describe(
                'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
              )
          )
          .describe('Period duration for the recurrence')
          .describe('The unit of time for the interval.'),
      })
      .describe('Recurring period with an interval and an anchor.')
      .optional()
      .describe('The subject of the grant.'),
  })
  .describe('The grant creation input.')

/**
 * Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided subject-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.

This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
 * @summary Override entitlement
 */
export const overrideEntitlementParams = zod.object({
  entitlementIdOrFeatureKey: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const overrideEntitlementBodyFeatureKeyMax = 64

export const overrideEntitlementBodyFeatureKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const overrideEntitlementBodyFeatureIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const overrideEntitlementBodyIsSoftLimitDefault = false
export const overrideEntitlementBodyIsUnlimitedDefault = false
export const overrideEntitlementBodyIssueAfterResetMin = 0
export const overrideEntitlementBodyIssueAfterResetPriorityDefault = 1
export const overrideEntitlementBodyIssueAfterResetPriorityMax = 255
export const overrideEntitlementBodyPreserveOverageAtResetDefault = false
export const overrideEntitlementBodyFeatureKeyMaxOne = 64

export const overrideEntitlementBodyFeatureKeyRegExpOne = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const overrideEntitlementBodyFeatureIdRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const overrideEntitlementBodyFeatureKeyMaxTwo = 64

export const overrideEntitlementBodyFeatureKeyRegExpTwo = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const overrideEntitlementBodyFeatureIdRegExpTwo = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const overrideEntitlementBody = zod
  .discriminatedUnion('type', [
    zod
      .object({
        featureId: zod
          .string()
          .regex(overrideEntitlementBodyFeatureIdRegExp)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(overrideEntitlementBodyFeatureKeyMax)
          .regex(overrideEntitlementBodyFeatureKeyRegExp)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        isSoftLimit: zod
          .boolean()
          .optional()
          .describe(
            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
          ),
        issueAfterReset: zod
          .number()
          .min(overrideEntitlementBodyIssueAfterResetMin)
          .optional()
          .describe(
            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
          ),
        issueAfterResetPriority: zod
          .number()
          .min(1)
          .max(overrideEntitlementBodyIssueAfterResetPriorityMax)
          .default(overrideEntitlementBodyIssueAfterResetPriorityDefault)
          .describe('Defines the grant priority for the default grant.'),
        isUnlimited: zod
          .boolean()
          .optional()
          .describe(
            'Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed in the future.'
          ),
        measureUsageFrom: zod
          .enum(['CURRENT_PERIOD_START', 'NOW'])
          .describe('Start of measurement options')
          .or(
            zod
              .date()
              .describe(
                '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
              )
          )
          .describe('Measure usage from')
          .optional()
          .describe(
            'Defines the time from which usage is measured. If not specified on creation, defaults to entitlement creation time.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        preserveOverageAtReset: zod
          .boolean()
          .optional()
          .describe(
            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
          ),
        type: zod.enum(['metered']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inpurs for metered entitlement'),
    zod
      .object({
        config: zod
          .string()
          .describe(
            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
          ),
        featureId: zod
          .string()
          .regex(overrideEntitlementBodyFeatureIdRegExpOne)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(overrideEntitlementBodyFeatureKeyMaxOne)
          .regex(overrideEntitlementBodyFeatureKeyRegExpOne)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        type: zod.enum(['static']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .optional()
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inputs for static entitlement'),
    zod
      .object({
        featureId: zod
          .string()
          .regex(overrideEntitlementBodyFeatureIdRegExpTwo)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        featureKey: zod
          .string()
          .min(1)
          .max(overrideEntitlementBodyFeatureKeyMaxTwo)
          .regex(overrideEntitlementBodyFeatureKeyRegExpTwo)
          .optional()
          .describe(
            'The feature the subject is entitled to use.\nEither featureKey or featureId is required.'
          ),
        metadata: zod
          .record(zod.string(), zod.string())
          .describe(
            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
          )
          .optional()
          .describe('Additional metadata for the feature.'),
        type: zod.enum(['boolean']),
        usagePeriod: zod
          .object({
            anchor: zod
              .date()
              .optional()
              .describe('A date-time anchor to base the recurring period on.'),
            interval: zod
              .string()
              .or(
                zod
                  .enum(['DAY', 'WEEK', 'MONTH', 'YEAR'])
                  .describe(
                    'The unit of time for the interval.\nOne of: `day`, `week`, `month`, or `year`.'
                  )
              )
              .describe('Period duration for the recurrence')
              .describe('The unit of time for the interval.'),
          })
          .describe('Recurring period with an interval and an anchor.')
          .optional()
          .describe('The usage period associated with the entitlement.'),
      })
      .describe('Create inputs for boolean entitlement'),
  ])
  .describe('Create inputs for entitlement')

/**
 * This endpoint should be used for access checks and enforcement. All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.

For convenience reasons, /value works with both entitlementId and featureKey.
 * @summary Get entitlement value
 */
export const getEntitlementValueParams = zod.object({
  entitlementIdOrFeatureKey: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const getEntitlementValueQueryParams = zod.object({
  time: zod.date().optional(),
})

/**
 * Get entitlement by id. For checking entitlement access, use the /value endpoint instead.
 * @summary Get entitlement
 */
export const getEntitlementParams = zod.object({
  entitlementId: zod.string(),
  subjectIdOrKey: zod.string(),
})

/**
 * Deleting an entitlement revokes access to the associated feature. As a single subject can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
 * @summary Delete entitlement
 */
export const deleteEntitlementParams = zod.object({
  entitlementId: zod.string(),
  subjectIdOrKey: zod.string(),
})

/**
 * Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.

BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.

WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
 * @summary Get entitlement history
 */
export const getEntitlementHistoryParams = zod.object({
  entitlementId: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const getEntitlementHistoryQueryWindowTimeZoneDefault = 'UTC'

export const getEntitlementHistoryQueryParams = zod.object({
  from: zod
    .date()
    .optional()
    .describe(
      'Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter.'
    ),
  to: zod
    .date()
    .optional()
    .describe(
      'End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now.\nIf not now then gets truncated to the granularity of the underlying meter.'
    ),
  windowSize: zod.enum(['MINUTE', 'HOUR', 'DAY']).describe('Windowsize'),
  windowTimeZone: zod
    .string()
    .default(getEntitlementHistoryQueryWindowTimeZoneDefault)
    .describe('The timezone used when calculating the windows.'),
})

/**
 * Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.

Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
 * @summary Reset entitlement
 */
export const resetEntitlementUsageParams = zod.object({
  entitlementId: zod.string(),
  subjectIdOrKey: zod.string(),
})

export const resetEntitlementUsageBody = zod
  .object({
    effectiveAt: zod
      .date()
      .optional()
      .describe(
        'The time at which the reset takes effect, defaults to now. The reset cannot be in the future. The provided value is truncated to the minute due to how historical meter data is stored.'
      ),
    preserveOverage: zod
      .boolean()
      .optional()
      .describe(
        "Determines whether the overage is preserved or forgiven, overriding the entitlement's default behavior.\n- If true, the overage is preserved.\n- If false, the overage is forgiven."
      ),
    retainAnchor: zod
      .boolean()
      .optional()
      .describe(
        'Determines whether the usage period anchor is retained or reset to the effectiveAt time.\n- If true, the usage period anchor is retained.\n- If false, the usage period anchor is reset to the effectiveAt time.'
      ),
  })
  .describe('Reset parameters')

/**
 * @summary Create subscription
 */
export const createSubscriptionBodyPlanKeyMax = 64

export const createSubscriptionBodyPlanKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createSubscriptionBodyTimingDefault = 'immediate'
export const createSubscriptionBodyCustomerIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createSubscriptionBodyCustomerKeyMax = 256
export const createSubscriptionBodyCustomPlanNameMax = 256
export const createSubscriptionBodyCustomPlanDescriptionMax = 1024
export const createSubscriptionBodyCustomPlanCurrencyMinOne = 3

export const createSubscriptionBodyCustomPlanCurrencyMaxOne = 3

export const createSubscriptionBodyCustomPlanCurrencyRegExpOne = new RegExp(
  '^[A-Z]{3}$'
)
export const createSubscriptionBodyCustomPlanCurrencyDefault = 'USD'
export const createSubscriptionBodyCustomPlanProRatingConfigEnabledDefault =
  true
export const createSubscriptionBodyCustomPlanProRatingConfigModeDefault =
  'prorate_prices'
export const createSubscriptionBodyCustomPlanProRatingConfigDefault = {
  enabled: true,
  mode: 'prorate_prices',
}
export const createSubscriptionBodyCustomPlanPhasesItemKeyMax = 64

export const createSubscriptionBodyCustomPlanPhasesItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const createSubscriptionBodyCustomPlanPhasesItemNameMax = 256
export const createSubscriptionBodyCustomPlanPhasesItemDescriptionMax = 1024
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMax = 64

export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMax = 256
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMax = 1024
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMax = 64

export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefault =
  'in_advance'
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMaxOne = 64

export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMaxOne = 256
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMaxOne = 1024
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMaxOne = 64

export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierDefault =
  '1'
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const createSubscriptionBodyTimingDefaultFour = 'immediate'
export const createSubscriptionBodyCustomerIdRegExpOne = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const createSubscriptionBodyCustomerKeyMaxOne = 256

export const createSubscriptionBody = zod
  .object({
    alignment: zod
      .object({
        billablesMustAlign: zod
          .boolean()
          .optional()
          .describe(
            "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
          ),
      })
      .describe('Alignment configuration for a plan or subscription.')
      .optional()
      .describe('What alignment settings the subscription should have.'),
    billingAnchor: zod
      .date()
      .optional()
      .describe(
        'The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.'
      ),
    customerId: zod
      .string()
      .regex(createSubscriptionBodyCustomerIdRegExp)
      .optional()
      .describe(
        'The ID of the customer. Provide either the key or ID. Has presedence over the key.'
      ),
    customerKey: zod
      .string()
      .min(1)
      .max(createSubscriptionBodyCustomerKeyMax)
      .optional()
      .describe('The key of the customer. Provide either the key or ID.'),
    description: zod
      .string()
      .optional()
      .describe('Description for the Subscription.'),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional()
      .describe('Arbitrary metadata associated with the subscription.'),
    name: zod
      .string()
      .optional()
      .describe(
        'The name of the Subscription. If not provided the plan name is used.'
      ),
    plan: zod
      .object({
        key: zod
          .string()
          .min(1)
          .max(createSubscriptionBodyPlanKeyMax)
          .regex(createSubscriptionBodyPlanKeyRegExp)
          .describe('The plan key.'),
        version: zod.number().optional().describe('The plan version.'),
      })
      .describe(
        'References an exact plan defaulting to the current active version.'
      )
      .describe('The plan reference to change to.'),
    startingPhase: zod
      .string()
      .min(1)
      .optional()
      .describe(
        'The key of the phase to start the subscription in.\nIf not provided, the subscription will start in the first phase of the plan.'
      ),
    timing: zod
      .enum(['immediate', 'next_billing_cycle'])
      .describe(
        'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
      )
      .or(
        zod
          .date()
          .describe(
            '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
          )
      )
      .describe(
        'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
      )
      .default(createSubscriptionBodyTimingDefault)
      .describe(
        'Timing configuration for the change, when the change should take effect.\nThe default is immediate.'
      ),
  })
  .describe('Create subscription based on plan.')
  .or(
    zod
      .object({
        billingAnchor: zod
          .date()
          .optional()
          .describe(
            'The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.'
          ),
        customerId: zod
          .string()
          .regex(createSubscriptionBodyCustomerIdRegExpOne)
          .optional()
          .describe(
            'The ID of the customer. Provide either the key or ID. Has presedence over the key.'
          ),
        customerKey: zod
          .string()
          .min(1)
          .max(createSubscriptionBodyCustomerKeyMaxOne)
          .optional()
          .describe('The key of the customer. Provide either the key or ID.'),
        customPlan: zod
          .object({
            alignment: zod
              .object({
                billablesMustAlign: zod
                  .boolean()
                  .optional()
                  .describe(
                    "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
                  ),
              })
              .describe('Alignment configuration for a plan or subscription.')
              .optional()
              .describe('Alignment configuration for the plan.'),
            billingCadence: zod
              .string()
              .describe(
                'The default billing cadence for subscriptions using this plan.\nDefines how often customers are billed using ISO8601 duration format.\nExamples: \"P1M\" (monthly), \"P3M\" (quarterly), \"P1Y\" (annually).'
              ),
            currency: zod
              .string()
              .min(createSubscriptionBodyCustomPlanCurrencyMinOne)
              .max(createSubscriptionBodyCustomPlanCurrencyMaxOne)
              .regex(createSubscriptionBodyCustomPlanCurrencyRegExpOne)
              .describe(
                'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
              )
              .describe('The currency code of the plan.'),
            description: zod
              .string()
              .max(createSubscriptionBodyCustomPlanDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(createSubscriptionBodyCustomPlanNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            phases: zod
              .array(
                zod
                  .object({
                    description: zod
                      .string()
                      .max(
                        createSubscriptionBodyCustomPlanPhasesItemDescriptionMax
                      )
                      .optional()
                      .describe(
                        'Optional description of the resource. Maximum 1024 characters.'
                      ),
                    duration: zod
                      .string()
                      .nullable()
                      .describe('The duration of the phase.'),
                    key: zod
                      .string()
                      .min(1)
                      .max(createSubscriptionBodyCustomPlanPhasesItemKeyMax)
                      .regex(
                        createSubscriptionBodyCustomPlanPhasesItemKeyRegExp
                      )
                      .describe('A semi-unique identifier for the resource.'),
                    metadata: zod
                      .record(zod.string(), zod.string())
                      .describe(
                        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                      )
                      .nullish()
                      .describe('Additional metadata for the resource.'),
                    name: zod
                      .string()
                      .min(1)
                      .max(createSubscriptionBodyCustomPlanPhasesItemNameMax)
                      .describe(
                        'Human-readable name for the resource. Between 1 and 256 characters.'
                      ),
                    rateCards: zod
                      .array(
                        zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                billingCadence: zod
                                  .string()
                                  .nullable()
                                  .describe(
                                    'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                                  ),
                                description: zod
                                  .string()
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMax
                                  )
                                  .optional()
                                  .describe(
                                    'Optional description of the resource. Maximum 1024 characters.'
                                  ),
                                discounts: zod
                                  .object({
                                    percentage: zod
                                      .object({
                                        percentage: zod
                                          .number()
                                          .describe(
                                            'Numeric representation of a percentage\n\n50% is represented as 50'
                                          )
                                          .describe(
                                            'The percentage of the discount.'
                                          ),
                                      })
                                      .describe('Percentage discount.')
                                      .optional()
                                      .describe('The percentage discount.'),
                                    usage: zod
                                      .object({
                                        quantity: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity of the usage discount.\n\nMust be positive.'
                                          ),
                                      })
                                      .describe(
                                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                                      )
                                      .optional()
                                      .describe('The usage discount.'),
                                  })
                                  .describe('Discount by type on a price')
                                  .optional()
                                  .describe(
                                    'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                                  ),
                                entitlementTemplate: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        isSoftLimit: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                          ),
                                        issueAfterReset: zod
                                          .number()
                                          .min(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin
                                          )
                                          .optional()
                                          .describe(
                                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                          ),
                                        issueAfterResetPriority: zod
                                          .number()
                                          .min(1)
                                          .max(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                                          )
                                          .default(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                                          )
                                          .describe(
                                            'Defines the grant priority for the default grant.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        preserveOverageAtReset: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                          ),
                                        type: zod.enum(['metered']),
                                        usagePeriod: zod
                                          .string()
                                          .optional()
                                          .describe(
                                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                          ),
                                      })
                                      .describe(
                                        'The entitlement template with a metered entitlement.'
                                      ),
                                    zod
                                      .object({
                                        config: zod
                                          .string()
                                          .describe(
                                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['static']),
                                      })
                                      .describe(
                                        'Entitlement template of a static entitlement.'
                                      ),
                                    zod
                                      .object({
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['boolean']),
                                      })
                                      .describe(
                                        'Entitlement template of a boolean entitlement.'
                                      ),
                                  ])
                                  .describe(
                                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                                  )
                                  .optional()
                                  .describe(
                                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                                  ),
                                featureKey: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMax
                                  )
                                  .regex(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExp
                                  )
                                  .optional()
                                  .describe(
                                    'The feature the customer is entitled to use.'
                                  ),
                                key: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMax
                                  )
                                  .regex(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExp
                                  )
                                  .describe(
                                    'A semi-unique identifier for the resource.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .nullish()
                                  .describe(
                                    'Additional metadata for the resource.'
                                  ),
                                name: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMax
                                  )
                                  .describe(
                                    'Human-readable name for the resource. Between 1 and 256 characters.'
                                  ),
                                price: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    paymentTerm: zod
                                      .enum(['in_advance', 'in_arrears'])
                                      .describe(
                                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                      )
                                      .default(
                                        createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefault
                                      )
                                      .describe(
                                        'The payment term of the flat price.\nDefaults to in advance.'
                                      ),
                                    type: zod.enum(['flat']),
                                  })
                                  .describe('Flat price with payment term.')
                                  .nullable()
                                  .describe(
                                    'The price of the rate card.\nWhen null, the feature or service is free.'
                                  ),
                                taxConfig: zod
                                  .object({
                                    behavior: zod
                                      .enum(['inclusive', 'exclusive'])
                                      .describe(
                                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                                      )
                                      .optional()
                                      .describe(
                                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                                      ),
                                    customInvoicing: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .describe(
                                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                          ),
                                      })
                                      .describe('Custom invoicing tax config.')
                                      .optional()
                                      .describe('Custom invoicing tax config.'),
                                    stripe: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExp
                                          )
                                          .describe(
                                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                          ),
                                      })
                                      .describe('The tax config for Stripe.')
                                      .optional()
                                      .describe('Stripe tax config.'),
                                  })
                                  .describe(
                                    'Set of provider specific tax configs.'
                                  )
                                  .optional()
                                  .describe(
                                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                                  ),
                                type: zod.enum(['flat_fee']),
                              })
                              .describe(
                                'A flat fee rate card defines a one-time purchase or a recurring fee.'
                              ),
                            zod
                              .object({
                                billingCadence: zod
                                  .string()
                                  .describe(
                                    'The billing cadence of the rate card.'
                                  ),
                                description: zod
                                  .string()
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMaxOne
                                  )
                                  .optional()
                                  .describe(
                                    'Optional description of the resource. Maximum 1024 characters.'
                                  ),
                                discounts: zod
                                  .object({
                                    percentage: zod
                                      .object({
                                        percentage: zod
                                          .number()
                                          .describe(
                                            'Numeric representation of a percentage\n\n50% is represented as 50'
                                          )
                                          .describe(
                                            'The percentage of the discount.'
                                          ),
                                      })
                                      .describe('Percentage discount.')
                                      .optional()
                                      .describe('The percentage discount.'),
                                    usage: zod
                                      .object({
                                        quantity: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity of the usage discount.\n\nMust be positive.'
                                          ),
                                      })
                                      .describe(
                                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                                      )
                                      .optional()
                                      .describe('The usage discount.'),
                                  })
                                  .describe('Discount by type on a price')
                                  .optional()
                                  .describe(
                                    'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                                  ),
                                entitlementTemplate: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        isSoftLimit: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                          ),
                                        issueAfterReset: zod
                                          .number()
                                          .min(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                                          )
                                          .optional()
                                          .describe(
                                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                          ),
                                        issueAfterResetPriority: zod
                                          .number()
                                          .min(1)
                                          .max(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                                          )
                                          .default(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                                          )
                                          .describe(
                                            'Defines the grant priority for the default grant.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        preserveOverageAtReset: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                          ),
                                        type: zod.enum(['metered']),
                                        usagePeriod: zod
                                          .string()
                                          .optional()
                                          .describe(
                                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                          ),
                                      })
                                      .describe(
                                        'The entitlement template with a metered entitlement.'
                                      ),
                                    zod
                                      .object({
                                        config: zod
                                          .string()
                                          .describe(
                                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['static']),
                                      })
                                      .describe(
                                        'Entitlement template of a static entitlement.'
                                      ),
                                    zod
                                      .object({
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['boolean']),
                                      })
                                      .describe(
                                        'Entitlement template of a boolean entitlement.'
                                      ),
                                  ])
                                  .describe(
                                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                                  )
                                  .optional()
                                  .describe(
                                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                                  ),
                                featureKey: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMaxOne
                                  )
                                  .regex(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExpOne
                                  )
                                  .optional()
                                  .describe(
                                    'The feature the customer is entitled to use.'
                                  ),
                                key: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMaxOne
                                  )
                                  .regex(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExpOne
                                  )
                                  .describe(
                                    'A semi-unique identifier for the resource.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .nullish()
                                  .describe(
                                    'Additional metadata for the resource.'
                                  ),
                                name: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMaxOne
                                  )
                                  .describe(
                                    'Human-readable name for the resource. Between 1 and 256 characters.'
                                  ),
                                price: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The amount of the flat price.'
                                          ),
                                        paymentTerm: zod
                                          .enum(['in_advance', 'in_arrears'])
                                          .describe(
                                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                          )
                                          .default(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefaultTwo
                                          )
                                          .describe(
                                            'The payment term of the flat price.\nDefaults to in advance.'
                                          ),
                                        type: zod.enum(['flat']),
                                      })
                                      .describe(
                                        'Flat price with payment term.'
                                      ),
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The amount of the unit price.'
                                          ),
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        type: zod.enum(['unit']),
                                      })
                                      .describe(
                                        'Unit price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        mode: zod
                                          .enum(['volume', 'graduated'])
                                          .describe(
                                            'The mode of the tiered price.'
                                          )
                                          .describe(
                                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                                          ),
                                        tiers: zod
                                          .array(
                                            zod
                                              .object({
                                                flatPrice: zod
                                                  .object({
                                                    amount: zod
                                                      .string()
                                                      .regex(
                                                        createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                                      )
                                                      .describe(
                                                        'Numeric represents an arbitrary precision number.'
                                                      )
                                                      .describe(
                                                        'The amount of the flat price.'
                                                      ),
                                                    type: zod
                                                      .enum(['flat'])
                                                      .describe(
                                                        'The type of the price.'
                                                      ),
                                                  })
                                                  .describe('Flat price.')
                                                  .nullable()
                                                  .describe(
                                                    'The flat price component of the tier.'
                                                  ),
                                                unitPrice: zod
                                                  .object({
                                                    amount: zod
                                                      .string()
                                                      .regex(
                                                        createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                                      )
                                                      .describe(
                                                        'Numeric represents an arbitrary precision number.'
                                                      )
                                                      .describe(
                                                        'The amount of the unit price.'
                                                      ),
                                                    type: zod
                                                      .enum(['unit'])
                                                      .describe(
                                                        'The type of the price.'
                                                      ),
                                                  })
                                                  .describe('Unit price.')
                                                  .nullable()
                                                  .describe(
                                                    'The unit price component of the tier.'
                                                  ),
                                                upToAmount: zod
                                                  .string()
                                                  .regex(
                                                    createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                                  )
                                                  .describe(
                                                    'Numeric represents an arbitrary precision number.'
                                                  )
                                                  .optional()
                                                  .describe(
                                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                                  ),
                                              })
                                              .describe(
                                                'A price tier.\nAt least one price component is required in each tier.'
                                              )
                                          )
                                          .min(1)
                                          .describe(
                                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                                          ),
                                        type: zod.enum(['tiered']),
                                      })
                                      .describe(
                                        'Tiered price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        multiplier: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .default(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierDefault
                                          )
                                          .describe(
                                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                                          ),
                                        type: zod.enum(['dynamic']),
                                      })
                                      .describe(
                                        'Dynamic price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The price of one package.'
                                          ),
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        quantityPerPackage: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity per package.'
                                          ),
                                        type: zod.enum(['package']),
                                      })
                                      .describe(
                                        'Package price with spend commitments.'
                                      ),
                                  ])
                                  .describe(
                                    'The price of the usage based rate card.'
                                  )
                                  .nullable()
                                  .describe(
                                    'The price of the rate card.\nWhen null, the feature or service is free.'
                                  ),
                                taxConfig: zod
                                  .object({
                                    behavior: zod
                                      .enum(['inclusive', 'exclusive'])
                                      .describe(
                                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                                      )
                                      .optional()
                                      .describe(
                                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                                      ),
                                    customInvoicing: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .describe(
                                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                          ),
                                      })
                                      .describe('Custom invoicing tax config.')
                                      .optional()
                                      .describe('Custom invoicing tax config.'),
                                    stripe: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .regex(
                                            createSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne
                                          )
                                          .describe(
                                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                          ),
                                      })
                                      .describe('The tax config for Stripe.')
                                      .optional()
                                      .describe('Stripe tax config.'),
                                  })
                                  .describe(
                                    'Set of provider specific tax configs.'
                                  )
                                  .optional()
                                  .describe(
                                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                                  ),
                                type: zod.enum(['usage_based']),
                              })
                              .describe(
                                'A usage-based rate card defines a price based on usage.'
                              ),
                          ])
                          .describe(
                            'A rate card defines the pricing and entitlement of a feature or service.'
                          )
                      )
                      .describe('The rate cards of the plan.'),
                  })
                  .describe(
                    "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses."
                  )
              )
              .min(1)
              .describe(
                "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.\nA phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices."
              ),
            proRatingConfig: zod
              .object({
                enabled: zod
                  .boolean()
                  .describe('Whether pro-rating is enabled for this plan.'),
                mode: zod
                  .enum(['prorate_prices'])
                  .describe(
                    'Pro-rating mode options for handling billing period changes.'
                  )
                  .describe(
                    'How to handle pro-rating for billing period changes.'
                  ),
              })
              .describe('Configuration for pro-rating behavior.')
              .default(createSubscriptionBodyCustomPlanProRatingConfigDefault)
              .describe(
                'Default pro-rating configuration for subscriptions using this plan.'
              ),
          })
          .describe('The template for omitting properties.')
          .describe(
            'Plan input for custom subscription creation (without key and version).'
          )
          .describe(
            'The custom plan description which defines the Subscription.'
          ),
        timing: zod
          .enum(['immediate', 'next_billing_cycle'])
          .describe(
            'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
          )
          .or(
            zod
              .date()
              .describe(
                '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
              )
          )
          .describe(
            'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
          )
          .default(createSubscriptionBodyTimingDefaultFour)
          .describe(
            'Timing configuration for the change, when the change should take effect.\nThe default is immediate.'
          ),
      })
      .describe('Create a custom subscription.')
  )
  .describe('Create a subscription.')

/**
 * @summary Get subscription
 */
export const getSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getSubscriptionParams = zod.object({
  subscriptionId: zod.string().regex(getSubscriptionPathSubscriptionIdRegExp),
})

export const getSubscriptionQueryParams = zod.object({
  at: zod
    .date()
    .optional()
    .describe(
      'The time at which the subscription should be queried. If not provided the current time is used.'
    ),
})

/**
 * Batch processing commands for manipulating running subscriptions.
The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
 * @summary Edit subscription
 */
export const editSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const editSubscriptionParams = zod.object({
  subscriptionId: zod.string().regex(editSubscriptionPathSubscriptionIdRegExp),
})

export const editSubscriptionBodyCustomizationsItemRateCardKeyMax = 64

export const editSubscriptionBodyCustomizationsItemRateCardKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const editSubscriptionBodyCustomizationsItemRateCardNameMax = 256
export const editSubscriptionBodyCustomizationsItemRateCardDescriptionMax = 1024
export const editSubscriptionBodyCustomizationsItemRateCardFeatureKeyMax = 64

export const editSubscriptionBodyCustomizationsItemRateCardFeatureKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIsSoftLimitDefault =
  false
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetMin = 0
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityMax = 255
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const editSubscriptionBodyCustomizationsItemRateCardTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPricePaymentTermDefault =
  'in_advance'
export const editSubscriptionBodyCustomizationsItemRateCardDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardKeyMaxOne = 64

export const editSubscriptionBodyCustomizationsItemRateCardKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const editSubscriptionBodyCustomizationsItemRateCardNameMaxOne = 256
export const editSubscriptionBodyCustomizationsItemRateCardDescriptionMaxOne = 1024
export const editSubscriptionBodyCustomizationsItemRateCardFeatureKeyMaxOne = 64

export const editSubscriptionBodyCustomizationsItemRateCardFeatureKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetMinOne = 0
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const editSubscriptionBodyCustomizationsItemRateCardTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPricePaymentTermDefaultTwo =
  'in_advance'
export const editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMultiplierDefault =
  '1'
export const editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemRateCardDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemPhaseDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const editSubscriptionBodyCustomizationsItemPhaseKeyMaxTwo = 64

export const editSubscriptionBodyCustomizationsItemPhaseKeyRegExpTwo =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const editSubscriptionBodyCustomizationsMax = 100

export const editSubscriptionBody = zod
  .object({
    customizations: zod
      .array(
        zod
          .discriminatedUnion('op', [
            zod
              .object({
                op: zod.enum(['add_item']),
                phaseKey: zod.string(),
                rateCard: zod
                  .discriminatedUnion('type', [
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .nullable()
                          .describe(
                            'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                          ),
                        description: zod
                          .string()
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardDescriptionMax
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardDiscountsUsageQuantityRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetMin
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityMax
                                  )
                                  .default(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityDefault
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardFeatureKeyMax
                          )
                          .regex(
                            editSubscriptionBodyCustomizationsItemRateCardFeatureKeyRegExp
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardKeyMax
                          )
                          .regex(
                            editSubscriptionBodyCustomizationsItemRateCardKeyRegExp
                          )
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardNameMax
                          )
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .object({
                            amount: zod
                              .string()
                              .regex(
                                editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .describe('The amount of the flat price.'),
                            paymentTerm: zod
                              .enum(['in_advance', 'in_arrears'])
                              .describe(
                                'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                              )
                              .default(
                                editSubscriptionBodyCustomizationsItemRateCardPricePaymentTermDefault
                              )
                              .describe(
                                'The payment term of the flat price.\nDefaults to in advance.'
                              ),
                            type: zod.enum(['flat']),
                          })
                          .describe('Flat price with payment term.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardTaxConfigStripeCodeRegExp
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['flat_fee']),
                      })
                      .describe(
                        'A flat fee rate card defines a one-time purchase or a recurring fee.'
                      ),
                    zod
                      .object({
                        billingCadence: zod
                          .string()
                          .describe('The billing cadence of the rate card.'),
                        description: zod
                          .string()
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardDescriptionMaxOne
                          )
                          .optional()
                          .describe(
                            'Optional description of the resource. Maximum 1024 characters.'
                          ),
                        discounts: zod
                          .object({
                            percentage: zod
                              .object({
                                percentage: zod
                                  .number()
                                  .describe(
                                    'Numeric representation of a percentage\n\n50% is represented as 50'
                                  )
                                  .describe('The percentage of the discount.'),
                              })
                              .describe('Percentage discount.')
                              .optional()
                              .describe('The percentage discount.'),
                            usage: zod
                              .object({
                                quantity: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardDiscountsUsageQuantityRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe(
                                    'The quantity of the usage discount.\n\nMust be positive.'
                                  ),
                              })
                              .describe(
                                'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                              )
                              .optional()
                              .describe('The usage discount.'),
                          })
                          .describe('Discount by type on a price')
                          .optional()
                          .describe(
                            'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                          ),
                        entitlementTemplate: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                isSoftLimit: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                  ),
                                issueAfterReset: zod
                                  .number()
                                  .min(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetMinOne
                                  )
                                  .optional()
                                  .describe(
                                    'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                  ),
                                issueAfterResetPriority: zod
                                  .number()
                                  .min(1)
                                  .max(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityMaxOne
                                  )
                                  .default(
                                    editSubscriptionBodyCustomizationsItemRateCardEntitlementTemplateIssueAfterResetPriorityDefaultOne
                                  )
                                  .describe(
                                    'Defines the grant priority for the default grant.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                preserveOverageAtReset: zod
                                  .boolean()
                                  .optional()
                                  .describe(
                                    'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                  ),
                                type: zod.enum(['metered']),
                                usagePeriod: zod
                                  .string()
                                  .optional()
                                  .describe(
                                    'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                  ),
                              })
                              .describe(
                                'The entitlement template with a metered entitlement.'
                              ),
                            zod
                              .object({
                                config: zod
                                  .string()
                                  .describe(
                                    'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['static']),
                              })
                              .describe(
                                'Entitlement template of a static entitlement.'
                              ),
                            zod
                              .object({
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .optional()
                                  .describe(
                                    'Additional metadata for the feature.'
                                  ),
                                type: zod.enum(['boolean']),
                              })
                              .describe(
                                'Entitlement template of a boolean entitlement.'
                              ),
                          ])
                          .describe(
                            'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                          )
                          .optional()
                          .describe(
                            'The entitlement of the rate card.\nOnly available when featureKey is set.'
                          ),
                        featureKey: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardFeatureKeyMaxOne
                          )
                          .regex(
                            editSubscriptionBodyCustomizationsItemRateCardFeatureKeyRegExpOne
                          )
                          .optional()
                          .describe(
                            'The feature the customer is entitled to use.'
                          ),
                        key: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardKeyMaxOne
                          )
                          .regex(
                            editSubscriptionBodyCustomizationsItemRateCardKeyRegExpOne
                          )
                          .describe(
                            'A semi-unique identifier for the resource.'
                          ),
                        metadata: zod
                          .record(zod.string(), zod.string())
                          .describe(
                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                          )
                          .nullish()
                          .describe('Additional metadata for the resource.'),
                        name: zod
                          .string()
                          .min(1)
                          .max(
                            editSubscriptionBodyCustomizationsItemRateCardNameMaxOne
                          )
                          .describe(
                            'Human-readable name for the resource. Between 1 and 256 characters.'
                          ),
                        price: zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the flat price.'),
                                paymentTerm: zod
                                  .enum(['in_advance', 'in_arrears'])
                                  .describe(
                                    'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                  )
                                  .default(
                                    editSubscriptionBodyCustomizationsItemRateCardPricePaymentTermDefaultTwo
                                  )
                                  .describe(
                                    'The payment term of the flat price.\nDefaults to in advance.'
                                  ),
                                type: zod.enum(['flat']),
                              })
                              .describe('Flat price with payment term.'),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The amount of the unit price.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                type: zod.enum(['unit']),
                              })
                              .describe('Unit price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpThree
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                mode: zod
                                  .enum(['volume', 'graduated'])
                                  .describe('The mode of the tiered price.')
                                  .describe(
                                    'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                                  ),
                                tiers: zod
                                  .array(
                                    zod
                                      .object({
                                        flatPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemFlatPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the flat price.'
                                              ),
                                            type: zod
                                              .enum(['flat'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Flat price.')
                                          .nullable()
                                          .describe(
                                            'The flat price component of the tier.'
                                          ),
                                        unitPrice: zod
                                          .object({
                                            amount: zod
                                              .string()
                                              .regex(
                                                editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemUnitPriceAmountRegExpOne
                                              )
                                              .describe(
                                                'Numeric represents an arbitrary precision number.'
                                              )
                                              .describe(
                                                'The amount of the unit price.'
                                              ),
                                            type: zod
                                              .enum(['unit'])
                                              .describe(
                                                'The type of the price.'
                                              ),
                                          })
                                          .describe('Unit price.')
                                          .nullable()
                                          .describe(
                                            'The unit price component of the tier.'
                                          ),
                                        upToAmount: zod
                                          .string()
                                          .regex(
                                            editSubscriptionBodyCustomizationsItemRateCardPriceTiersItemUpToAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                          ),
                                      })
                                      .describe(
                                        'A price tier.\nAt least one price component is required in each tier.'
                                      )
                                  )
                                  .min(1)
                                  .describe(
                                    'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                                  ),
                                type: zod.enum(['tiered']),
                              })
                              .describe('Tiered price with spend commitments.'),
                            zod
                              .object({
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpFive
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                multiplier: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMultiplierRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .default(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMultiplierDefault
                                  )
                                  .describe(
                                    'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                                  ),
                                type: zod.enum(['dynamic']),
                              })
                              .describe(
                                'Dynamic price with spend commitments.'
                              ),
                            zod
                              .object({
                                amount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The price of one package.'),
                                maximumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMaximumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is limited to spend at most the amount.'
                                  ),
                                minimumAmount: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceMinimumAmountRegExpSeven
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .optional()
                                  .describe(
                                    'The customer is committed to spend at least the amount.'
                                  ),
                                quantityPerPackage: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardPriceQuantityPerPackageRegExpOne
                                  )
                                  .describe(
                                    'Numeric represents an arbitrary precision number.'
                                  )
                                  .describe('The quantity per package.'),
                                type: zod.enum(['package']),
                              })
                              .describe(
                                'Package price with spend commitments.'
                              ),
                          ])
                          .describe('The price of the usage based rate card.')
                          .nullable()
                          .describe(
                            'The price of the rate card.\nWhen null, the feature or service is free.'
                          ),
                        taxConfig: zod
                          .object({
                            behavior: zod
                              .enum(['inclusive', 'exclusive'])
                              .describe(
                                'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                              )
                              .optional()
                              .describe(
                                "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                              ),
                            customInvoicing: zod
                              .object({
                                code: zod
                                  .string()
                                  .describe(
                                    'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                  ),
                              })
                              .describe('Custom invoicing tax config.')
                              .optional()
                              .describe('Custom invoicing tax config.'),
                            stripe: zod
                              .object({
                                code: zod
                                  .string()
                                  .regex(
                                    editSubscriptionBodyCustomizationsItemRateCardTaxConfigStripeCodeRegExpOne
                                  )
                                  .describe(
                                    'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                  ),
                              })
                              .describe('The tax config for Stripe.')
                              .optional()
                              .describe('Stripe tax config.'),
                          })
                          .describe('Set of provider specific tax configs.')
                          .optional()
                          .describe(
                            'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                          ),
                        type: zod.enum(['usage_based']),
                      })
                      .describe(
                        'A usage-based rate card defines a price based on usage.'
                      ),
                  ])
                  .describe(
                    'A rate card defines the pricing and entitlement of a feature or service.'
                  ),
              })
              .describe('Add a new item to a phase.'),
            zod
              .object({
                itemKey: zod.string(),
                op: zod.enum(['remove_item']),
                phaseKey: zod.string(),
              })
              .describe('Remove an item from a phase.'),
            zod
              .object({
                op: zod.enum(['add_phase']),
                phase: zod
                  .object({
                    description: zod
                      .string()
                      .optional()
                      .describe('The description of the phase.'),
                    discounts: zod
                      .object({
                        percentage: zod
                          .object({
                            percentage: zod
                              .number()
                              .describe(
                                'Numeric representation of a percentage\n\n50% is represented as 50'
                              )
                              .describe('The percentage of the discount.'),
                          })
                          .describe('Percentage discount.')
                          .optional()
                          .describe('The percentage discount.'),
                        usage: zod
                          .object({
                            quantity: zod
                              .string()
                              .regex(
                                editSubscriptionBodyCustomizationsItemPhaseDiscountsUsageQuantityRegExpOne
                              )
                              .describe(
                                'Numeric represents an arbitrary precision number.'
                              )
                              .describe(
                                'The quantity of the usage discount.\n\nMust be positive.'
                              ),
                          })
                          .describe(
                            'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                          )
                          .optional()
                          .describe('The usage discount.'),
                      })
                      .describe('Discount by type on a price')
                      .optional()
                      .describe('The discounts on the plan.'),
                    duration: zod
                      .string()
                      .optional()
                      .describe(
                        'The intended duration of the new phase.\nDuration is required when the phase will not be the last phase.'
                      ),
                    key: zod
                      .string()
                      .min(1)
                      .max(editSubscriptionBodyCustomizationsItemPhaseKeyMaxTwo)
                      .regex(
                        editSubscriptionBodyCustomizationsItemPhaseKeyRegExpTwo
                      )
                      .describe('A locally unique identifier for the phase.'),
                    name: zod.string().describe('The name of the phase.'),
                    startAfter: zod
                      .string()
                      .nullable()
                      .describe(
                        'Interval after the subscription starts to transition to the phase.\nWhen null, the phase starts immediately after the subscription starts.'
                      ),
                  })
                  .describe('Subscription phase create input.'),
              })
              .describe('Add a new phase'),
            zod
              .object({
                op: zod.enum(['remove_phase']),
                phaseKey: zod.string(),
                shift: zod
                  .enum(['next', 'prev'])
                  .describe(
                    'The direction of the phase shift when a phase is removed.'
                  ),
              })
              .describe('Remove a phase'),
            zod
              .object({
                extendBy: zod.string(),
                op: zod.enum(['stretch_phase']),
                phaseKey: zod.string(),
              })
              .describe('Stretch a phase'),
            zod
              .object({
                op: zod.enum(['unschedule_edit']),
              })
              .describe('Unschedules any edits from the current phase.'),
          ])
          .describe('The operation to be performed on the subscription.')
      )
      .max(editSubscriptionBodyCustomizationsMax)
      .describe(
        'Batch processing commands for manipulating running subscriptions.\nThe key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.'
      ),
    timing: zod
      .enum(['immediate', 'next_billing_cycle'])
      .describe(
        'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
      )
      .or(
        zod
          .date()
          .describe(
            '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
          )
      )
      .describe(
        'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
      )
      .optional()
      .describe(
        'Whether the billing period should be restarted.Timing configuration to allow for the changes to take effect at different times.'
      ),
  })
  .describe('Subscription edit input.')

/**
 * Deletes a subscription. Only scheduled subscriptions can be deleted.
 * @summary Delete subscription
 */
export const deleteSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const deleteSubscriptionParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(deleteSubscriptionPathSubscriptionIdRegExp),
})

/**
 * Create a new subscription addon, either providing the key or the id of the addon.
 * @summary Create subscription addon
 */
export const createSubscriptionAddonPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createSubscriptionAddonParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(createSubscriptionAddonPathSubscriptionIdRegExp),
})

export const createSubscriptionAddonBodyNameMax = 256
export const createSubscriptionAddonBodyDescriptionMax = 1024
export const createSubscriptionAddonBodyQuantityMin = 0
export const createSubscriptionAddonBodyAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const createSubscriptionAddonBody = zod
  .object({
    addon: zod
      .object({
        id: zod
          .string()
          .regex(createSubscriptionAddonBodyAddonIdRegExp)
          .describe('The ID of the add-on.'),
      })
      .describe('The add-on to create.'),
    description: zod
      .string()
      .max(createSubscriptionAddonBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(createSubscriptionAddonBodyNameMax)
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    quantity: zod
      .number()
      .min(createSubscriptionAddonBodyQuantityMin)
      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.'
      ),
    timing: zod
      .enum(['immediate', 'next_billing_cycle'])
      .describe(
        'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
      )
      .or(
        zod
          .date()
          .describe(
            '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
          )
      )
      .describe(
        'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
      )
      .describe(
        'The timing of the operation. After the create or update, a new entry will be created in the timeline.'
      ),
  })
  .describe('A subscription add-on create body.')

/**
 * List all addons of a subscription. In the returned list will match to a set unique by addonId.
 * @summary List subscription addons
 */
export const listSubscriptionAddonsPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const listSubscriptionAddonsParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(listSubscriptionAddonsPathSubscriptionIdRegExp),
})

/**
 * Get a subscription addon by id.
 * @summary Get subscription addon
 */
export const getSubscriptionAddonPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const getSubscriptionAddonPathSubscriptionAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const getSubscriptionAddonParams = zod.object({
  subscriptionAddonId: zod
    .string()
    .regex(getSubscriptionAddonPathSubscriptionAddonIdRegExp),
  subscriptionId: zod
    .string()
    .regex(getSubscriptionAddonPathSubscriptionIdRegExp),
})

/**
 * Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)
 * @summary Update subscription addon
 */
export const updateSubscriptionAddonPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)
export const updateSubscriptionAddonPathSubscriptionAddonIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const updateSubscriptionAddonParams = zod.object({
  subscriptionAddonId: zod
    .string()
    .regex(updateSubscriptionAddonPathSubscriptionAddonIdRegExp),
  subscriptionId: zod
    .string()
    .regex(updateSubscriptionAddonPathSubscriptionIdRegExp),
})

export const updateSubscriptionAddonBodyNameMax = 256
export const updateSubscriptionAddonBodyDescriptionMax = 1024
export const updateSubscriptionAddonBodyQuantityMin = 0

export const updateSubscriptionAddonBody = zod
  .object({
    description: zod
      .string()
      .max(updateSubscriptionAddonBodyDescriptionMax)
      .optional()
      .describe(
        'Optional description of the resource. Maximum 1024 characters.'
      ),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .nullish()
      .describe('Additional metadata for the resource.'),
    name: zod
      .string()
      .min(1)
      .max(updateSubscriptionAddonBodyNameMax)
      .optional()
      .describe(
        'Human-readable name for the resource. Between 1 and 256 characters.'
      ),
    quantity: zod
      .number()
      .min(updateSubscriptionAddonBodyQuantityMin)
      .optional()
      .describe(
        'The quantity of the add-on. Always 1 for single instance add-ons.'
      ),
    timing: zod
      .enum(['immediate', 'next_billing_cycle'])
      .describe(
        'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
      )
      .or(
        zod
          .date()
          .describe(
            '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
          )
      )
      .describe(
        'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
      )
      .optional()
      .describe(
        'The timing of the operation. After the create or update, a new entry will be created in the timeline.'
      ),
  })
  .describe('Resource create or update operation model.')

/**
 * Cancels the subscription.
Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.
 * @summary Cancel subscription
 */
export const cancelSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const cancelSubscriptionParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(cancelSubscriptionPathSubscriptionIdRegExp),
})

export const cancelSubscriptionBody = zod.object({
  timing: zod
    .enum(['immediate', 'next_billing_cycle'])
    .describe(
      'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
    )
    .or(
      zod
        .date()
        .describe(
          '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
        )
    )
    .describe(
      'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
    )
    .optional()
    .describe('If not provided the subscription is canceled immediately.'),
})

/**
 * Closes a running subscription and starts a new one according to the specification.
Can be used for upgrades, downgrades, and plan changes.
 * @summary Change subscription
 */
export const changeSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const changeSubscriptionParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(changeSubscriptionPathSubscriptionIdRegExp),
})

export const changeSubscriptionBodyPlanKeyMax = 64

export const changeSubscriptionBodyPlanKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const changeSubscriptionBodyCustomPlanNameMax = 256
export const changeSubscriptionBodyCustomPlanDescriptionMax = 1024
export const changeSubscriptionBodyCustomPlanCurrencyMinOne = 3

export const changeSubscriptionBodyCustomPlanCurrencyMaxOne = 3

export const changeSubscriptionBodyCustomPlanCurrencyRegExpOne = new RegExp(
  '^[A-Z]{3}$'
)
export const changeSubscriptionBodyCustomPlanCurrencyDefault = 'USD'
export const changeSubscriptionBodyCustomPlanProRatingConfigEnabledDefault =
  true
export const changeSubscriptionBodyCustomPlanProRatingConfigModeDefault =
  'prorate_prices'
export const changeSubscriptionBodyCustomPlanProRatingConfigDefault = {
  enabled: true,
  mode: 'prorate_prices',
}
export const changeSubscriptionBodyCustomPlanPhasesItemKeyMax = 64

export const changeSubscriptionBodyCustomPlanPhasesItemKeyRegExp = new RegExp(
  '^[a-z0-9]+(?:_[a-z0-9]+)*$'
)
export const changeSubscriptionBodyCustomPlanPhasesItemNameMax = 256
export const changeSubscriptionBodyCustomPlanPhasesItemDescriptionMax = 1024
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMax = 64

export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMax = 256
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMax = 1024
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMax = 64

export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExp =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefault =
  false
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin = 0
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault = 1
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax = 255
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefault =
  false
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExp =
  new RegExp('^txcd_\\d{8}$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefault =
  'in_advance'
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMaxOne = 64

export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMaxOne = 256
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMaxOne = 1024
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMaxOne = 64

export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExpOne =
  new RegExp('^[a-z0-9]+(?:_[a-z0-9]+)*$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIsSoftLimitDefaultOne =
  false
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne = 0
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne = 1
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne = 255
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplatePreserveOverageAtResetDefaultOne =
  false
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne =
  new RegExp('^txcd_\\d{8}$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefaultTwo =
  'in_advance'
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierDefault =
  '1'
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpFive =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')
export const changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree =
  new RegExp('^\\-?[0-9]+(\\.[0-9]+)?$')

export const changeSubscriptionBody = zod
  .object({
    alignment: zod
      .object({
        billablesMustAlign: zod
          .boolean()
          .optional()
          .describe(
            "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
          ),
      })
      .describe('Alignment configuration for a plan or subscription.')
      .optional()
      .describe('What alignment settings the subscription should have.'),
    description: zod
      .string()
      .optional()
      .describe('Description for the Subscription.'),
    metadata: zod
      .record(zod.string(), zod.string())
      .describe(
        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
      )
      .optional()
      .describe('Arbitrary metadata associated with the subscription.'),
    name: zod
      .string()
      .optional()
      .describe(
        'The name of the Subscription. If not provided the plan name is used.'
      ),
    plan: zod
      .object({
        key: zod
          .string()
          .min(1)
          .max(changeSubscriptionBodyPlanKeyMax)
          .regex(changeSubscriptionBodyPlanKeyRegExp)
          .describe('The plan key.'),
        version: zod.number().optional().describe('The plan version.'),
      })
      .describe(
        'References an exact plan defaulting to the current active version.'
      )
      .describe('The plan reference to change to.'),
    startingPhase: zod
      .string()
      .min(1)
      .optional()
      .describe(
        'The key of the phase to start the subscription in.\nIf not provided, the subscription will start in the first phase of the plan.'
      ),
    timing: zod
      .enum(['immediate', 'next_billing_cycle'])
      .describe(
        'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
      )
      .or(
        zod
          .date()
          .describe(
            '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
          )
      )
      .describe(
        'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
      )
      .describe(
        'Timing configuration for the change, when the change should take effect.\nFor changing a subscription, the accepted values depend on the subscription configuration.'
      ),
  })
  .describe('Change subscription based on plan.')
  .or(
    zod
      .object({
        customPlan: zod
          .object({
            alignment: zod
              .object({
                billablesMustAlign: zod
                  .boolean()
                  .optional()
                  .describe(
                    "Whether all Billable items and RateCards must align.\nAlignment means the Price's BillingCadence must align for both duration and anchor time."
                  ),
              })
              .describe('Alignment configuration for a plan or subscription.')
              .optional()
              .describe('Alignment configuration for the plan.'),
            billingCadence: zod
              .string()
              .describe(
                'The default billing cadence for subscriptions using this plan.\nDefines how often customers are billed using ISO8601 duration format.\nExamples: \"P1M\" (monthly), \"P3M\" (quarterly), \"P1Y\" (annually).'
              ),
            currency: zod
              .string()
              .min(changeSubscriptionBodyCustomPlanCurrencyMinOne)
              .max(changeSubscriptionBodyCustomPlanCurrencyMaxOne)
              .regex(changeSubscriptionBodyCustomPlanCurrencyRegExpOne)
              .describe(
                'Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.\nCustom three-letter currency codes are also supported for convenience.'
              )
              .describe('The currency code of the plan.'),
            description: zod
              .string()
              .max(changeSubscriptionBodyCustomPlanDescriptionMax)
              .optional()
              .describe(
                'Optional description of the resource. Maximum 1024 characters.'
              ),
            metadata: zod
              .record(zod.string(), zod.string())
              .describe(
                'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
              )
              .nullish()
              .describe('Additional metadata for the resource.'),
            name: zod
              .string()
              .min(1)
              .max(changeSubscriptionBodyCustomPlanNameMax)
              .describe(
                'Human-readable name for the resource. Between 1 and 256 characters.'
              ),
            phases: zod
              .array(
                zod
                  .object({
                    description: zod
                      .string()
                      .max(
                        changeSubscriptionBodyCustomPlanPhasesItemDescriptionMax
                      )
                      .optional()
                      .describe(
                        'Optional description of the resource. Maximum 1024 characters.'
                      ),
                    duration: zod
                      .string()
                      .nullable()
                      .describe('The duration of the phase.'),
                    key: zod
                      .string()
                      .min(1)
                      .max(changeSubscriptionBodyCustomPlanPhasesItemKeyMax)
                      .regex(
                        changeSubscriptionBodyCustomPlanPhasesItemKeyRegExp
                      )
                      .describe('A semi-unique identifier for the resource.'),
                    metadata: zod
                      .record(zod.string(), zod.string())
                      .describe(
                        'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                      )
                      .nullish()
                      .describe('Additional metadata for the resource.'),
                    name: zod
                      .string()
                      .min(1)
                      .max(changeSubscriptionBodyCustomPlanPhasesItemNameMax)
                      .describe(
                        'Human-readable name for the resource. Between 1 and 256 characters.'
                      ),
                    rateCards: zod
                      .array(
                        zod
                          .discriminatedUnion('type', [
                            zod
                              .object({
                                billingCadence: zod
                                  .string()
                                  .nullable()
                                  .describe(
                                    'The billing cadence of the rate card.\nWhen null it means it is a one time fee.'
                                  ),
                                description: zod
                                  .string()
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMax
                                  )
                                  .optional()
                                  .describe(
                                    'Optional description of the resource. Maximum 1024 characters.'
                                  ),
                                discounts: zod
                                  .object({
                                    percentage: zod
                                      .object({
                                        percentage: zod
                                          .number()
                                          .describe(
                                            'Numeric representation of a percentage\n\n50% is represented as 50'
                                          )
                                          .describe(
                                            'The percentage of the discount.'
                                          ),
                                      })
                                      .describe('Percentage discount.')
                                      .optional()
                                      .describe('The percentage discount.'),
                                    usage: zod
                                      .object({
                                        quantity: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity of the usage discount.\n\nMust be positive.'
                                          ),
                                      })
                                      .describe(
                                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                                      )
                                      .optional()
                                      .describe('The usage discount.'),
                                  })
                                  .describe('Discount by type on a price')
                                  .optional()
                                  .describe(
                                    'The discount of the rate card. For flat fee rate cards only percentage discounts are supported.\nOnly available when price is set.'
                                  ),
                                entitlementTemplate: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        isSoftLimit: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                          ),
                                        issueAfterReset: zod
                                          .number()
                                          .min(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMin
                                          )
                                          .optional()
                                          .describe(
                                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                          ),
                                        issueAfterResetPriority: zod
                                          .number()
                                          .min(1)
                                          .max(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMax
                                          )
                                          .default(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefault
                                          )
                                          .describe(
                                            'Defines the grant priority for the default grant.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        preserveOverageAtReset: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                          ),
                                        type: zod.enum(['metered']),
                                        usagePeriod: zod
                                          .string()
                                          .optional()
                                          .describe(
                                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                          ),
                                      })
                                      .describe(
                                        'The entitlement template with a metered entitlement.'
                                      ),
                                    zod
                                      .object({
                                        config: zod
                                          .string()
                                          .describe(
                                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['static']),
                                      })
                                      .describe(
                                        'Entitlement template of a static entitlement.'
                                      ),
                                    zod
                                      .object({
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['boolean']),
                                      })
                                      .describe(
                                        'Entitlement template of a boolean entitlement.'
                                      ),
                                  ])
                                  .describe(
                                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                                  )
                                  .optional()
                                  .describe(
                                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                                  ),
                                featureKey: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMax
                                  )
                                  .regex(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExp
                                  )
                                  .optional()
                                  .describe(
                                    'The feature the customer is entitled to use.'
                                  ),
                                key: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMax
                                  )
                                  .regex(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExp
                                  )
                                  .describe(
                                    'A semi-unique identifier for the resource.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .nullish()
                                  .describe(
                                    'Additional metadata for the resource.'
                                  ),
                                name: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMax
                                  )
                                  .describe(
                                    'Human-readable name for the resource. Between 1 and 256 characters.'
                                  ),
                                price: zod
                                  .object({
                                    amount: zod
                                      .string()
                                      .regex(
                                        changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpOne
                                      )
                                      .describe(
                                        'Numeric represents an arbitrary precision number.'
                                      )
                                      .describe(
                                        'The amount of the flat price.'
                                      ),
                                    paymentTerm: zod
                                      .enum(['in_advance', 'in_arrears'])
                                      .describe(
                                        'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                      )
                                      .default(
                                        changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefault
                                      )
                                      .describe(
                                        'The payment term of the flat price.\nDefaults to in advance.'
                                      ),
                                    type: zod.enum(['flat']),
                                  })
                                  .describe('Flat price with payment term.')
                                  .nullable()
                                  .describe(
                                    'The price of the rate card.\nWhen null, the feature or service is free.'
                                  ),
                                taxConfig: zod
                                  .object({
                                    behavior: zod
                                      .enum(['inclusive', 'exclusive'])
                                      .describe(
                                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                                      )
                                      .optional()
                                      .describe(
                                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                                      ),
                                    customInvoicing: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .describe(
                                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                          ),
                                      })
                                      .describe('Custom invoicing tax config.')
                                      .optional()
                                      .describe('Custom invoicing tax config.'),
                                    stripe: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExp
                                          )
                                          .describe(
                                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                          ),
                                      })
                                      .describe('The tax config for Stripe.')
                                      .optional()
                                      .describe('Stripe tax config.'),
                                  })
                                  .describe(
                                    'Set of provider specific tax configs.'
                                  )
                                  .optional()
                                  .describe(
                                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                                  ),
                                type: zod.enum(['flat_fee']),
                              })
                              .describe(
                                'A flat fee rate card defines a one-time purchase or a recurring fee.'
                              ),
                            zod
                              .object({
                                billingCadence: zod
                                  .string()
                                  .describe(
                                    'The billing cadence of the rate card.'
                                  ),
                                description: zod
                                  .string()
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDescriptionMaxOne
                                  )
                                  .optional()
                                  .describe(
                                    'Optional description of the resource. Maximum 1024 characters.'
                                  ),
                                discounts: zod
                                  .object({
                                    percentage: zod
                                      .object({
                                        percentage: zod
                                          .number()
                                          .describe(
                                            'Numeric representation of a percentage\n\n50% is represented as 50'
                                          )
                                          .describe(
                                            'The percentage of the discount.'
                                          ),
                                      })
                                      .describe('Percentage discount.')
                                      .optional()
                                      .describe('The percentage discount.'),
                                    usage: zod
                                      .object({
                                        quantity: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemDiscountsUsageQuantityRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity of the usage discount.\n\nMust be positive.'
                                          ),
                                      })
                                      .describe(
                                        'Usage discount.\n\nUsage discount means that the first N items are free. From billing perspective\nthis means that any usage on a specific feature is considered 0 until this discount\nis exhausted.'
                                      )
                                      .optional()
                                      .describe('The usage discount.'),
                                  })
                                  .describe('Discount by type on a price')
                                  .optional()
                                  .describe(
                                    'The discounts of the rate card.\n\nFlat fee rate cards only support percentage discounts.'
                                  ),
                                entitlementTemplate: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        isSoftLimit: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.'
                                          ),
                                        issueAfterReset: zod
                                          .number()
                                          .min(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetMinOne
                                          )
                                          .optional()
                                          .describe(
                                            'You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.\nIf an amount is specified here, a grant will be created alongside the entitlement with the specified amount.\nThat grant will have it\'s rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.\nManually creating such a grant would mean having the \"amount\", \"minRolloverAmount\", and \"maxRolloverAmount\" fields all be the same.'
                                          ),
                                        issueAfterResetPriority: zod
                                          .number()
                                          .min(1)
                                          .max(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityMaxOne
                                          )
                                          .default(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemEntitlementTemplateIssueAfterResetPriorityDefaultOne
                                          )
                                          .describe(
                                            'Defines the grant priority for the default grant.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        preserveOverageAtReset: zod
                                          .boolean()
                                          .optional()
                                          .describe(
                                            'If true, the overage is preserved at reset. If false, the usage is reset to 0.'
                                          ),
                                        type: zod.enum(['metered']),
                                        usagePeriod: zod
                                          .string()
                                          .optional()
                                          .describe(
                                            'The interval of the metered entitlement.\nDefaults to the billing cadence of the rate card.'
                                          ),
                                      })
                                      .describe(
                                        'The entitlement template with a metered entitlement.'
                                      ),
                                    zod
                                      .object({
                                        config: zod
                                          .string()
                                          .describe(
                                            'The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.'
                                          ),
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['static']),
                                      })
                                      .describe(
                                        'Entitlement template of a static entitlement.'
                                      ),
                                    zod
                                      .object({
                                        metadata: zod
                                          .record(zod.string(), zod.string())
                                          .describe(
                                            'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                          )
                                          .optional()
                                          .describe(
                                            'Additional metadata for the feature.'
                                          ),
                                        type: zod.enum(['boolean']),
                                      })
                                      .describe(
                                        'Entitlement template of a boolean entitlement.'
                                      ),
                                  ])
                                  .describe(
                                    'Entitlement templates are used to define the entitlements of a plan.\nFeatures are omitted from the entitlement template, as they are defined in the rate card.'
                                  )
                                  .optional()
                                  .describe(
                                    'The entitlement of the rate card.\nOnly available when featureKey is set.'
                                  ),
                                featureKey: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyMaxOne
                                  )
                                  .regex(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemFeatureKeyRegExpOne
                                  )
                                  .optional()
                                  .describe(
                                    'The feature the customer is entitled to use.'
                                  ),
                                key: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyMaxOne
                                  )
                                  .regex(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemKeyRegExpOne
                                  )
                                  .describe(
                                    'A semi-unique identifier for the resource.'
                                  ),
                                metadata: zod
                                  .record(zod.string(), zod.string())
                                  .describe(
                                    'Set of key-value pairs.\nMetadata can be used to store additional information about a resource.'
                                  )
                                  .nullish()
                                  .describe(
                                    'Additional metadata for the resource.'
                                  ),
                                name: zod
                                  .string()
                                  .min(1)
                                  .max(
                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemNameMaxOne
                                  )
                                  .describe(
                                    'Human-readable name for the resource. Between 1 and 256 characters.'
                                  ),
                                price: zod
                                  .discriminatedUnion('type', [
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The amount of the flat price.'
                                          ),
                                        paymentTerm: zod
                                          .enum(['in_advance', 'in_arrears'])
                                          .describe(
                                            'The payment term of a flat price.\nOne of: in_advance or in_arrears.'
                                          )
                                          .default(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPricePaymentTermDefaultTwo
                                          )
                                          .describe(
                                            'The payment term of the flat price.\nDefaults to in advance.'
                                          ),
                                        type: zod.enum(['flat']),
                                      })
                                      .describe(
                                        'Flat price with payment term.'
                                      ),
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The amount of the unit price.'
                                          ),
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        type: zod.enum(['unit']),
                                      })
                                      .describe(
                                        'Unit price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpThree
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        mode: zod
                                          .enum(['volume', 'graduated'])
                                          .describe(
                                            'The mode of the tiered price.'
                                          )
                                          .describe(
                                            'Defines if the tiering mode is volume-based or graduated:\n- In `volume`-based tiering, the maximum quantity within a period determines the per unit price.\n- In `graduated` tiering, pricing can change as the quantity grows.'
                                          ),
                                        tiers: zod
                                          .array(
                                            zod
                                              .object({
                                                flatPrice: zod
                                                  .object({
                                                    amount: zod
                                                      .string()
                                                      .regex(
                                                        changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemFlatPriceAmountRegExpOne
                                                      )
                                                      .describe(
                                                        'Numeric represents an arbitrary precision number.'
                                                      )
                                                      .describe(
                                                        'The amount of the flat price.'
                                                      ),
                                                    type: zod
                                                      .enum(['flat'])
                                                      .describe(
                                                        'The type of the price.'
                                                      ),
                                                  })
                                                  .describe('Flat price.')
                                                  .nullable()
                                                  .describe(
                                                    'The flat price component of the tier.'
                                                  ),
                                                unitPrice: zod
                                                  .object({
                                                    amount: zod
                                                      .string()
                                                      .regex(
                                                        changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUnitPriceAmountRegExpOne
                                                      )
                                                      .describe(
                                                        'Numeric represents an arbitrary precision number.'
                                                      )
                                                      .describe(
                                                        'The amount of the unit price.'
                                                      ),
                                                    type: zod
                                                      .enum(['unit'])
                                                      .describe(
                                                        'The type of the price.'
                                                      ),
                                                  })
                                                  .describe('Unit price.')
                                                  .nullable()
                                                  .describe(
                                                    'The unit price component of the tier.'
                                                  ),
                                                upToAmount: zod
                                                  .string()
                                                  .regex(
                                                    changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceTiersItemUpToAmountRegExpOne
                                                  )
                                                  .describe(
                                                    'Numeric represents an arbitrary precision number.'
                                                  )
                                                  .optional()
                                                  .describe(
                                                    'Up to and including to this quantity will be contained in the tier.\nIf null, the tier is open-ended.'
                                                  ),
                                              })
                                              .describe(
                                                'A price tier.\nAt least one price component is required in each tier.'
                                              )
                                          )
                                          .min(1)
                                          .describe(
                                            'The tiers of the tiered price.\nAt least one price component is required in each tier.'
                                          ),
                                        type: zod.enum(['tiered']),
                                      })
                                      .describe(
                                        'Tiered price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpFive
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        multiplier: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .default(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMultiplierDefault
                                          )
                                          .describe(
                                            'The multiplier to apply to the base price to get the dynamic price.\n\nExamples:\n- 0.0: the price is zero\n- 0.5: the price is 50% of the base price\n- 1.0: the price is the same as the base price\n- 1.5: the price is 150% of the base price'
                                          ),
                                        type: zod.enum(['dynamic']),
                                      })
                                      .describe(
                                        'Dynamic price with spend commitments.'
                                      ),
                                    zod
                                      .object({
                                        amount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The price of one package.'
                                          ),
                                        maximumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMaximumAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is limited to spend at most the amount.'
                                          ),
                                        minimumAmount: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceMinimumAmountRegExpSeven
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .optional()
                                          .describe(
                                            'The customer is committed to spend at least the amount.'
                                          ),
                                        quantityPerPackage: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemPriceQuantityPerPackageRegExpOne
                                          )
                                          .describe(
                                            'Numeric represents an arbitrary precision number.'
                                          )
                                          .describe(
                                            'The quantity per package.'
                                          ),
                                        type: zod.enum(['package']),
                                      })
                                      .describe(
                                        'Package price with spend commitments.'
                                      ),
                                  ])
                                  .describe(
                                    'The price of the usage based rate card.'
                                  )
                                  .nullable()
                                  .describe(
                                    'The price of the rate card.\nWhen null, the feature or service is free.'
                                  ),
                                taxConfig: zod
                                  .object({
                                    behavior: zod
                                      .enum(['inclusive', 'exclusive'])
                                      .describe(
                                        'Tax behavior.\n\nThis enum is used to specify whether tax is included in the price or excluded from the price.'
                                      )
                                      .optional()
                                      .describe(
                                        "Tax behavior.\n\nIf not specified the billing profile is used to determine the tax behavior.\nIf not specified in the billing profile, the provider's default behavior is used."
                                      ),
                                    customInvoicing: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .describe(
                                            'Tax code.\n\nThe tax code should be interpreted by the custom invoicing provider.'
                                          ),
                                      })
                                      .describe('Custom invoicing tax config.')
                                      .optional()
                                      .describe('Custom invoicing tax config.'),
                                    stripe: zod
                                      .object({
                                        code: zod
                                          .string()
                                          .regex(
                                            changeSubscriptionBodyCustomPlanPhasesItemRateCardsItemTaxConfigStripeCodeRegExpOne
                                          )
                                          .describe(
                                            'Product tax code.\n\nSee: https://docs.stripe.com/tax/tax-codes'
                                          ),
                                      })
                                      .describe('The tax config for Stripe.')
                                      .optional()
                                      .describe('Stripe tax config.'),
                                  })
                                  .describe(
                                    'Set of provider specific tax configs.'
                                  )
                                  .optional()
                                  .describe(
                                    'The tax config of the rate card.\nWhen undefined, the tax config of the feature or the default tax config of the plan is used.'
                                  ),
                                type: zod.enum(['usage_based']),
                              })
                              .describe(
                                'A usage-based rate card defines a price based on usage.'
                              ),
                          ])
                          .describe(
                            'A rate card defines the pricing and entitlement of a feature or service.'
                          )
                      )
                      .describe('The rate cards of the plan.'),
                  })
                  .describe(
                    "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses."
                  )
              )
              .min(1)
              .describe(
                "The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.\nA phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices."
              ),
            proRatingConfig: zod
              .object({
                enabled: zod
                  .boolean()
                  .describe('Whether pro-rating is enabled for this plan.'),
                mode: zod
                  .enum(['prorate_prices'])
                  .describe(
                    'Pro-rating mode options for handling billing period changes.'
                  )
                  .describe(
                    'How to handle pro-rating for billing period changes.'
                  ),
              })
              .describe('Configuration for pro-rating behavior.')
              .default(changeSubscriptionBodyCustomPlanProRatingConfigDefault)
              .describe(
                'Default pro-rating configuration for subscriptions using this plan.'
              ),
          })
          .describe('The template for omitting properties.')
          .describe(
            'Plan input for custom subscription creation (without key and version).'
          )
          .describe(
            'The custom plan description which defines the Subscription.'
          ),
        timing: zod
          .enum(['immediate', 'next_billing_cycle'])
          .describe(
            'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
          )
          .or(
            zod
              .date()
              .describe(
                '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
              )
          )
          .describe(
            'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
          )
          .describe(
            'Timing configuration for the change, when the change should take effect.\nFor changing a subscription, the accepted values depend on the subscription configuration.'
          ),
      })
      .describe('Change a custom subscription.')
  )
  .describe('Change a subscription.')

/**
 * Migrates the subscripiton to the provided version of the current plan.
If possible, the migration will be done immediately.
If not, the migration will be scheduled to the end of the current billing period.
 * @summary Migrate subscription
 */
export const migrateSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const migrateSubscriptionParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(migrateSubscriptionPathSubscriptionIdRegExp),
})

export const migrateSubscriptionBodyTimingDefault = 'immediate'

export const migrateSubscriptionBody = zod.object({
  startingPhase: zod
    .string()
    .min(1)
    .optional()
    .describe(
      'The key of the phase to start the subscription in.\nIf not provided, the subscription will start in the first phase of the plan.'
    ),
  targetVersion: zod
    .number()
    .min(1)
    .optional()
    .describe(
      'The version of the plan to migrate to.\nIf not provided, the subscription will migrate to the latest version of the current plan.'
    ),
  timing: zod
    .enum(['immediate', 'next_billing_cycle'])
    .describe(
      'Subscription edit timing.\nWhen immediate, the requested changes take effect immediately.\nWhen nextBillingCycle, the requested changes take effect at the next billing cycle.'
    )
    .or(
      zod
        .date()
        .describe(
          '[RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.'
        )
    )
    .describe(
      'Subscription edit timing defined when the changes should take effect.\nIf the provided configuration is not supported by the subscription, an error will be returned.'
    )
    .default(migrateSubscriptionBodyTimingDefault)
    .describe(
      'Timing configuration for the migration, when the migration should take effect.\nIf not supported by the subscription, 400 will be returned.'
    ),
})

/**
 * Restores a canceled subscription.
Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.
 * @summary Restore subscription
 */
export const restoreSubscriptionPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const restoreSubscriptionParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(restoreSubscriptionPathSubscriptionIdRegExp),
})

/**
 * Cancels the scheduled cancelation.
 * @summary Unschedule cancelation
 */
export const unscheduleCancelationPathSubscriptionIdRegExp = new RegExp(
  '^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$'
)

export const unscheduleCancelationParams = zod.object({
  subscriptionId: zod
    .string()
    .regex(unscheduleCancelationPathSubscriptionIdRegExp),
})

/**
 * List ingested events with advanced filtering and cursor pagination.
 * @summary List ingested events
 */
export const listEventsV2QueryLimitDefault = 100
export const listEventsV2QueryLimitMax = 100
export const listEventsV2QueryClientIdMax = 36

export const listEventsV2QueryParams = zod.object({
  clientId: zod
    .string()
    .min(1)
    .max(listEventsV2QueryClientIdMax)
    .optional()
    .describe('Client ID\nUseful to track progress of a query.'),
  cursor: zod
    .string()
    .optional()
    .describe('The cursor after which to start the pagination.'),
  limit: zod
    .number()
    .min(1)
    .max(listEventsV2QueryLimitMax)
    .default(listEventsV2QueryLimitDefault)
    .describe('The limit of the pagination.'),
})
