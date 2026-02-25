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
  @extension("x-omitempty", true)
  $eq?: string | null;

  /**
   * The field must not be equal to the provided value.
   */
  @extension("x-omitempty", true)
  $ne?: string | null;

  /**
   * The field must be in the provided list of values.
   */
  @minItems(1)
  @maxItems(100)
  @extension("x-omitempty", true)
  $in?: string[] | null;

  /**
   * The field must not be in the provided list of values.
   */
  @minItems(1)
  @maxItems(100)
  @extension("x-omitempty", true)
  $nin?: string[] | null;

  /**
   * The field must match the provided value.
   */
  @extension("x-omitempty", true)
  $like?: string | null;

  /**
   * The field must not match the provided value.
   */
  @extension("x-omitempty", true)
  $nlike?: string | null;

  /**
   * The field must match the provided value, ignoring case.
   */
  @extension("x-omitempty", true)
  $ilike?: string | null;

  /**
   * The field must not match the provided value, ignoring case.
   */
  @extension("x-omitempty", true)
  $nilike?: string | null;

  /**
   * The field must be greater than the provided value.
   */
  @extension("x-omitempty", true)
  $gt?: string | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $gte?: string | null;

  /**
   * The field must be less than the provided value.
   */
  @extension("x-omitempty", true)
  $lt?: string | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $lte?: string | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $and?: FilterString[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $or?: FilterString[] | null;
}

/**
 * A filter for a ID (ULID) field allowing only equality or inclusion.
 */
#suppress "@openmeter/api-spec/casing" "ID"
@friendlyName("FilterIDExact")
model FilterIDExact {
  /**
   * The field must be in the provided list of values.
   */
  @minItems(1)
  @maxItems(100)
  @extension("x-omitempty", true)
  $in?: ULID[] | null;
}

/**
 * A filter for an integer field.
 */
@friendlyName("FilterInteger")
model FilterInteger {
  /**
   * The field must be equal to the provided value.
   */
  @extension("x-omitempty", true)
  $eq?: integer | null;

  /**
   * The field must not be equal to the provided value.
   */
  @extension("x-omitempty", true)
  $ne?: integer | null;

  /**
   * The field must be greater than the provided value.
   */
  @extension("x-omitempty", true)
  $gt?: integer | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $gte?: integer | null;

  /**
   * The field must be less than the provided value.
   */
  @extension("x-omitempty", true)
  $lt?: integer | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $lte?: integer | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $and?: FilterInteger[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
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
  @extension("x-omitempty", true)
  $eq?: float64 | null;

  /**
   * The field must not be equal to the provided value.
   */
  @extension("x-omitempty", true)
  $ne?: float64 | null;

  /**
   * The field must be greater than the provided value.
   */
  @extension("x-omitempty", true)
  $gt?: float64 | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $gte?: float64 | null;

  /**
   * The field must be less than the provided value.
   */
  @extension("x-omitempty", true)
  $lt?: float64 | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $lte?: float64 | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $and?: FilterFloat[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
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
  @extension("x-omitempty", true)
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
  @extension("x-omitempty", true)
  $gt?: utcDateTime | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $gte?: utcDateTime | null;

  /**
   * The field must be less than the provided value.
   */
  @extension("x-omitempty", true)
  $lt?: utcDateTime | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  @extension("x-omitempty", true)
  $lte?: utcDateTime | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $and?: FilterTime[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  @extension("x-omitempty", true)
  $or?: FilterTime[] | null;
}
