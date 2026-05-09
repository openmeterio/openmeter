import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/json-schema";
import "./properties.tsp";

using TypeSpec.JsonSchema;

namespace Shared;

/**
 * A query filter for a string attribute. Operators are mutually exclusive, only
 * one operator is allowed at a time.
 */
@friendlyName("QueryFilterString")
@summary("Query String Filter")
model QueryFilterString {
  /**
   * The attribute equals the provided value.
   */
  eq?: string;

  /**
   * The attribute does not equal the provided value.
   */
  neq?: string;

  /**
   * The attribute is one of the provided values.
   */
  @minItems(1)
  @maxItems(100)
  in?: string[];

  /**
   * The attribute is not one of the provided values.
   */
  @minItems(1)
  @maxItems(100)
  nin?: string[];

  /**
   * The attribute contains the provided value.
   */
  contains?: string;

  /**
   * The attribute does not contain the provided value.
   */
  ncontains?: string;

  /**
   * Combines the provided filters with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  and?: QueryFilterString[];

  /**
   * Combines the provided filters with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  or?: QueryFilterString[];
}

/**
 * A query filter for an item in a string map attribute. Operators are mutually
 * exclusive, only one operator is allowed at a time.
 */
@friendlyName("QueryFilterStringMapItem")
@summary("Query String Map Item Filter")
model QueryFilterStringMapItem {
  /**
   * The attribute exists.
   */
  exists?: boolean;

  ...QueryFilterString;
}

alias QueryFilterStringMap = Record<QueryFilterStringMapItem>;

/**
 * A query filter for an integer attribute. Operators are mutually exclusive, only
 * one operator is allowed at a time.
 */
@friendlyName("QueryFilterInteger")
@summary("Query Integer Filter")
model QueryFilterInteger {
  /**
   * The attribute equals the provided value.
   */
  eq?: integer;

  /**
   * The attribute does not equal the provided value.
   */
  neq?: integer;

  /**
   * The attribute is one of the provided values.
   */
  @minItems(1)
  @maxItems(100)
  in?: integer[];

  /**
   * The attribute is not one of the provided values.
   */
  @minItems(1)
  @maxItems(100)
  nin?: integer[];

  /**
   * The attribute is greater than the provided value.
   */
  gt?: integer;

  /**
   * The attribute is greater than or equal to the provided value.
   */
  gte?: integer;

  /**
   * The attribute is less than the provided value.
   */
  lt?: integer;

  /**
   * The attribute is less than or equal to the provided value.
   */
  lte?: integer;

  /**
   * Combines the provided filters with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  and?: QueryFilterInteger[];

  /**
   * Combines the provided filters with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  or?: QueryFilterInteger[];
}

/**
 * A query filter for a float attribute. Operators are mutually exclusive, only one
 * operator is allowed at a time.
 */
@friendlyName("QueryFilterFloat")
@summary("Query Float Filter")
model QueryFilterFloat {
  /**
   * The attribute is greater than the provided value.
   */
  gt?: float64;

  /**
   * The attribute is greater than or equal to the provided value.
   */
  gte?: float64;

  /**
   * The attribute is less than the provided value.
   */
  lt?: float64;

  /**
   * The attribute is less than or equal to the provided value.
   */
  lte?: float64;

  /**
   * Combines the provided filters with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  and?: QueryFilterFloat[];

  /**
   * Combines the provided filters with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  or?: QueryFilterFloat[];
}

/**
 * A query filter for a numeric attribute. Operators are mutually exclusive, only
 * one operator is allowed at a time.
 */
@friendlyName("QueryFilterNumeric")
@summary("Query Numeric Filter")
model QueryFilterNumeric {
  /**
   * The attribute is greater than the provided value.
   */
  gt?: Shared.Numeric;

  /**
   * The attribute is greater than or equal to the provided value.
   */
  gte?: Shared.Numeric;

  /**
   * The attribute is less than the provided value.
   */
  lt?: Shared.Numeric;

  /**
   * The attribute is less than or equal to the provided value.
   */
  lte?: Shared.Numeric;

  /**
   * Combines the provided filters with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  and?: QueryFilterNumeric[];

  /**
   * Combines the provided filters with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  or?: QueryFilterNumeric[];
}

/**
 * A query filter for a boolean attribute. Operators are mutually exclusive, only
 * one operator is allowed at a time.
 */
@friendlyName("QueryFilterBoolean")
@summary("Query Boolean Filter")
model QueryFilterBoolean {
  /**
   * The attribute equals the provided value.
   */
  eq?: boolean;
}

/**
 * A query filter for a time attribute. Operators are mutually exclusive, only one
 * operator is allowed at a time.
 */
@friendlyName("QueryFilterDateTime")
@summary("Query Date-time Filter")
model QueryFilterDateTime {
  /**
   * The attribute is greater than the provided value.
   */
  gt?: DateTime;

  /**
   * The attribute is greater than or equal to the provided value.
   */
  gte?: DateTime;

  /**
   * The attribute is less than the provided value.
   */
  lt?: DateTime;

  /**
   * The attribute is less than or equal to the provided value.
   */
  lte?: DateTime;

  /**
   * Combines the provided filters with a logical AND.
   */
  @minItems(1)
  @maxItems(10)
  and?: QueryFilterDateTime[];

  /**
   * Combines the provided filters with a logical OR.
   */
  @minItems(1)
  @maxItems(10)
  or?: QueryFilterDateTime[];
}
