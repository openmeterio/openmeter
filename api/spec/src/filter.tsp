import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * A filter for a string field.
 */
@friendlyName("FilterString")
model FilterString {
  /**
   * The field must be equal to the provided value.
   */
  $eq?: string | null;

  /**
   * The field must not be equal to the provided value.
   */
  $ne?: string | null;

  /**
   * The field must be in the provided list of values.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(100)
  $in?: string[] | null;

  /**
   * The field must not be in the provided list of values.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(100)
  $nin?: string[] | null;

  /**
   * The field must match the provided value.
   */
  $like?: string | null;

  /**
   * The field must not match the provided value.
   */
  $nlike?: string | null;

  /**
   * The field must match the provided value, ignoring case.
   */
  $ilike?: string | null;

  /**
   * The field must not match the provided value, ignoring case.
   */
  $nilike?: string | null;

  /**
   * The field must be greater than the provided value.
   */
  $gt?: string | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  $gte?: string | null;

  /**
   * The field must be less than the provided value.
   */
  $lt?: string | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  $lte?: string | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $and?: FilterString[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $or?: FilterString[] | null;
}

/**
 * A filter for an integer field.
 */
@friendlyName("FilterInteger")
model FilterInteger {
  /**
   * The field must be equal to the provided value.
   */
  $eq?: integer | null;

  /**
   * The field must not be equal to the provided value.
   */
  $ne?: integer | null;

  /**
   * The field must be greater than the provided value.
   */
  $gt?: integer | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  $gte?: integer | null;

  /**
   * The field must be less than the provided value.
   */
  $lt?: integer | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  $lte?: integer | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $and?: FilterInteger[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $or?: FilterInteger[] | null;
}

/**
 * A filter for a float field.
 */
@friendlyName("FilterFloat")
model FilterFloat {
  /**
   * The field must be equal to the provided value.
   */
  $eq?: float64 | null;

  /**
   * The field must not be equal to the provided value.
   */
  $ne?: float64 | null;

  /**
   * The field must be greater than the provided value.
   */
  $gt?: float64 | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  $gte?: float64 | null;

  /**
   * The field must be less than the provided value.
   */
  $lt?: float64 | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  $lte?: float64 | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $and?: FilterFloat[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $or?: FilterFloat[] | null;
}

/**
 * A filter for a boolean field.
 */
@friendlyName("FilterBoolean")
model FilterBoolean {
  /**
   * The field must be equal to the provided value.
   */
  $eq?: boolean | null;
}

/**
 * A filter for a time field.
 */
@friendlyName("FilterTime")
model FilterTime {
  /**
   * The field must be greater than the provided value.
   */
  $gt?: utcDateTime | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  $gte?: utcDateTime | null;

  /**
   * The field must be less than the provided value.
   */
  $lt?: utcDateTime | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  $lte?: utcDateTime | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $and?: FilterTime[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  // Not supported for nullable types
  // @minItems(1)
  // @maxItems(10)
  $or?: FilterTime[] | null;
}
