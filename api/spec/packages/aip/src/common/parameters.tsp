import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace Common;

/**
 * Sort query.
 *
 * The `asc` suffix is optional as the default sort order is ascending.
 * The `desc` suffix is used to specify a descending order.
 */
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/SortQuery")
@friendlyName("SortQuery")
model SortQuery {}

/**
 * Filter by a boolean value (true/false).
 */
@friendlyName("BooleanFieldFilter")
@extension("x-go-type", "filters.FilterBoolean")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("Boolean Field Filter")
union BooleanFieldFilter {
  @summary("Equals (short form)")
  equals: boolean,

  @summary("Equals")
  BooleanFieldEQFilter: {
    /** Value strictly equals the given boolean value. */
    eq: boolean,
  },
}

/**
 * Filter by a numeric value.
 * All properties are optional; provide exactly one to specify the comparison.
 */
@friendlyName("NumericFieldFilter")
@extension("x-go-type", "filters.FilterNumeric")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("Numeric Field Filter")
union NumericFieldFilter {
  @summary("Equals (short form)")
  equals: float64,

  @summary("Equals")
  NumericFieldEQFilter: {
    /** Value strictly equals the given numeric value. */
    eq: float64,
  },

  @summary("Not Equals")
  NumericFieldNEQFilter: {
    /** Value does not equal the given numeric value. */
    neq: float64,
  },

  @summary("Equals one of")
  NumericFieldOEQFilter: {
    /** Returns entities that match any of the comma-delimited numeric values. */
    @encode(ArrayEncoding.commaDelimited)
    oeq: float64[],
  },

  @summary("Less Than")
  NumericFieldLTFilter: {
    /** Value is less than the given numeric value. */
    lt: float64,
  },

  @summary("Less Than or Equal To")
  NumericFieldLTEFilter: {
    /** Value is less than or equal to the given numeric value. */
    lte: float64,
  },

  @summary("Greater Than")
  NumericFieldGTFilter: {
    /** Value is greater than the given numeric value. */
    gt: float64,
  },

  @summary("Greater Than or Equal To")
  NumericFieldGTEFilter: {
    /** Value is greater than or equal to the given numeric value. */
    gte: float64,
  },
}

/**
 * Filters on the given string field value by either exact or fuzzy match.
 * All properties are optional; provide exactly one to specify the comparison.
 */
@friendlyName("StringFieldFilter")
@extension("x-go-type", "filters.FilterString")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("String Field Filter")
union StringFieldFilter {
  @summary("Equals (short form)")
  equals: string,

  @summary("Equals")
  StringFieldEQFilter: {
    /** Value strictly equals the given string value. */
    eq: string,
  },

  @summary("Not Equals")
  StringFieldNEQFilter: {
    /** Value does not equal the given string value. */
    neq: string,
  },

  @summary("Contains")
  StringFieldContainsFilter: {
    /** Value contains the given string value (fuzzy match). */
    contains: string,
  },

  @summary("Contains one of")
  StringFieldOContainsFilter: {
    /** Returns entities that fuzzy-match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    ocontains: string[],
  },

  @summary("Equals one of")
  StringFieldOEQFilter: {
    /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq: string[],
  },

  @summary("Greater Than")
  StringFieldGTFilter: {
    /** Value is greater than the given string value (lexicographic compare). */
    gt: string,
  },

  @summary("Greater Than or Equal To")
  StringFieldGTEFilter: {
    /** Value is greater than or equal to the given string value (lexicographic compare). */
    gte: string,
  },

  @summary("Less Than")
  StringFieldLTFilter: {
    /** Value is less than the given string value (lexicographic compare). */
    lt: string,
  },

  @summary("Less Than or Equal To")
  StringFieldLTEFilter: {
    /** Value is less than or equal to the given string value (lexicographic compare). */
    lte: string,
  },

  @summary("Exists")
  StringFieldExistsFilter: {
    /** When true, the field must be present (non-null); when false, the field must be absent (null). */
    exists: boolean,
  },
}

/**
 * Filters on the given string field value by exact match.
 * All properties are optional; provide exactly one to specify the comparison.
 */
@friendlyName("StringFieldFilterExact")
@extension("x-go-type", "filters.FilterStringExact")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("String Field Filter Exact")
union StringFieldFilterExact {
  @summary("Equals (short form)")
  equals: string,

  @summary("Equals")
  StringFieldEQFilter: {
    /** Value strictly equals the given string value. */
    eq: string,
  },

  @summary("Equals one of")
  StringFieldOEQFilter: {
    /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq: string[],
  },

  @summary("Not Equals")
  StringFieldNEQFilter: {
    /** Value does not equal the given string value. */
    neq: string,
  },
}

/**
 * Filters on the given ULID field value by exact match.
 * All properties are optional; provide exactly one to specify the comparison.
 */
@friendlyName("ULIDFieldFilter")
@extension("x-go-type", "filters.FilterULID")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("ULID Field Filter")
union ULIDFieldFilter {
  @summary("Equals (short form)")
  equals: Shared.ULID,

  @summary("Equals")
  ULIDFieldEQFilter: {
    /** Value strictly equals the given ULID value. */
    eq: Shared.ULID,
  },

  @summary("Equals one of")
  ULIDFieldOEQFilter: {
    /** Returns entities that exact match any of the comma-delimited ULIDs in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq: Shared.ULID[],
  },

  @summary("Not Equals")
  ULIDFieldNEQFilter: {
    /** Value does not equal the given ULID value. */
    neq: Shared.ULID,
  },
}

/**
 * Filters on the given datetime (RFC-3339) field value.
 * All properties are optional; provide exactly one to specify the comparison.
 */
@friendlyName("DateTimeFieldFilter")
@extension("x-go-type", "filters.FilterDateTime")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("DateTime Field Filter")
union DateTimeFieldFilter {
  @summary("Equals (short form)")
  equals: Shared.DateTime,

  @summary("Equals")
  DateTimeFieldEQFilter: {
    /** Value strictly equals given RFC-3339 formatted timestamp in UTC. */
    eq: Shared.DateTime,
  },

  @summary("Less Than")
  DateTimeFieldLTFilter: {
    /** Value is less than the given RFC-3339 formatted timestamp in UTC. */
    lt: Shared.DateTime,
  },

  @summary("Less Than or Equal To")
  DateTimeFieldLTEFilter: {
    /** Value is less than or equal to the given RFC-3339 formatted timestamp in UTC. */
    lte: Shared.DateTime,
  },

  @summary("Greater Than")
  DateTimeFieldGTFilter: {
    /** Value is greater than the given RFC-3339 formatted timestamp in UTC. */
    gt: Shared.DateTime,
  },

  @summary("Greater Than or Equal To")
  DateTimeFieldGTEFilter: {
    /** Value is greater than or equal to the given RFC-3339 formatted timestamp in UTC. */
    gte: Shared.DateTime,
  },
}

/**
 * Filters on the resource's `labels` field.
 */
@friendlyName("LabelsFieldFilter")
@extension("x-go-type", "filters.FilterLabels")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
@oneOf
@summary("Labels Field Filter")
union LabelsFieldFilter {
  @summary("Equals (short form)")
  equals: string,

  @summary("Equals")
  LabelsFieldEqualsFilter: {
    /** Value strictly equals the given string value. */
    eq: string,
  },

  @summary("Contains")
  LabelsFieldContainsFilter: {
    /** Value contains the given string value (fuzzy match). */
    contains: string,
  },

  @summary("Contains one of")
  LabelsFieldOContainsFilter: {
    /** Returns entities that fuzzy-match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    ocontains: string[],
  },

  @summary("Equals one of")
  LabelsFieldOEQFilter: {
    /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq: string[],
  },

  @summary("Not Equals")
  LabelsFieldNEQFilter: {
    /** Value does not equal the given string value. */
    neq: string,
  },
}
