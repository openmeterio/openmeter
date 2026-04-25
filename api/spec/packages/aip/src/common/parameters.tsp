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
union BooleanFieldFilter {
  boolean: boolean,
  object: {
    /** Value strictly equals the given boolean value. */
    eq?: boolean,
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
model NumericFieldFilter {
  /** Value strictly equals the given numeric value. */
  eq?: float64;

  /** Value does not equal the given numeric value. */
  neq?: float64;

  /** Returns entities that match any of the comma-delimited numeric values. */
  @encode(ArrayEncoding.commaDelimited)
  oeq?: float64[];

  /** Value is less than the given numeric value. */
  lt?: float64;

  /** Value is less than or equal to the given numeric value. */
  lte?: float64;

  /** Value is greater than the given numeric value. */
  gt?: float64;

  /** Value is greater than or equal to the given numeric value. */
  gte?: float64;
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
union StringFieldFilter {
  string: string,
  object: {
    /** Value strictly equals the given string value. */
    eq?: string,

    /** Value does not equal the given string value. */
    neq?: string,

    /** Value contains the given string value (fuzzy match). */
    contains?: string,

    /** Returns entities that fuzzy-match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    ocontains?: string[],

    /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq?: string[],

    /** Value is greater than the given string value (lexicographic compare). */
    gt?: string,

    /** Value is greater than or equal to the given string value (lexicographic compare). */
    gte?: string,

    /** Value is less than the given string value (lexicographic compare). */
    lt?: string,

    /** Value is less than or equal to the given string value (lexicographic compare). */
    lte?: string,

    /** When true, the field must be present (non-null); when false, the field must be absent (null). */
    exists?: boolean,
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
model StringFieldFilterExact {
  /** Value strictly equals the given string value. */
  eq?: string;

  /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
  @encode(ArrayEncoding.commaDelimited)
  oeq?: string[];

  /** Value does not equal the given string value. */
  neq?: string;
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
union ULIDFieldFilter {
  string: Shared.ULID,
  object: {
    /** Value strictly equals the given ULID value. */
    eq?: Shared.ULID,

    /** Value does not equal the given ULID value. */
    neq?: Shared.ULID,

    /** Returns entities that exact match any of the comma-delimited ULIDs in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq?: Shared.ULID[],
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
union DateTimeFieldFilter {
  string: string,
  object: {
    /** Value strictly equals given RFC-3339 formatted timestamp in UTC. */
    eq?: Shared.DateTime,

    /** Value is less than the given RFC-3339 formatted timestamp in UTC. */
    lt?: Shared.DateTime,

    /** Value is less than or equal to the given RFC-3339 formatted timestamp in UTC. */
    lte?: Shared.DateTime,

    /** Value is greater than the given RFC-3339 formatted timestamp in UTC. */
    gt?: Shared.DateTime,

    /** Value is greater than or equal to the given RFC-3339 formatted timestamp in UTC. */
    gte?: Shared.DateTime,
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
union LabelsFieldFilter {
  string: string,
  object: {
    /** Value strictly equals the given string value. */
    eq?: string,

    /** Value contains the given string value (fuzzy match). */
    contains?: string,

    /** Returns entities that fuzzy-match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    ocontains?: string[],

    /** Returns entities that exact match any of the comma-delimited phrases in the filter string. */
    @encode(ArrayEncoding.commaDelimited)
    oeq?: string[],

    /** Value does not equal the given string value. */
    neq?: string,
  },
}
