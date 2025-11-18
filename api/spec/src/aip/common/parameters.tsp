import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace Common;

/**
 * Sort query.
 */
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/SortQuery")
@friendlyName("SortQuery")
model SortQuery {}

/**
 * Filter by a boolean value (true/false).
 */
@friendlyName("BooleanFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/BooleanFieldFilter")
model BooleanFieldFilter {}

/**
 * Filter by a numeric value.
 */
@friendlyName("NumericFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/NumericFieldFilter")
model NumericFieldFilter {}

/**
 * Filters on the given string field value by either exact or fuzzy match.
 */
@friendlyName("StringFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/StringFieldFilter")
model StringFieldFilter {}

/**
 * Filters on the given string field value by exact match.
 */
@friendlyName("StringFieldFilterExact")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/StringFieldFilterExact")
model StringFieldFilterExact {}

/**
 * Filters on the given datetime (RFC-3339) field value.
 */
@friendlyName("DateTimeFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/DateTimeFieldFilter")
model DateTimeFieldFilter {}

/**
 * Filters on the resource's `labels` field. Filters must use dot-notation to identify
 * the label key that will be used to filter the results. For example:
 *  - `filter[labels.owner]`
 *  - `filter[labels.owner][neq]=kong`
 *  - `filter[labels.env]=dev`
 *  - `filter[labels.env][ocontains]=dev,test`
 */
@friendlyName("LabelsFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/LabelsFieldFilter")
model LabelsFieldFilter {}

/**
 * Filters on the resource's `public_labels` field. Filters must use dot-notation to identify
 * the label key that will be used to filter the results. For example:
 *  - `filter[public_labels.owner]`
 *  - `filter[public_labels.owner][neq]=kong`
 *  - `filter[public_labels.env]=dev`
 *  - `filter[public_labels.env][ocontains]=dev,test`
 */
@friendlyName("PublicLabelsFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/PublicLabelsFieldFilter")
model PublicLabelsFieldFilter {}

/**
 * Filters on the resource's `attributes` field. Filters must use dot-notation to identify
 * the attribute key that will be used to filter the results. For example:
 *  - `filter[attributes.owner]`
 *  - `filter[attributes.owner][neq]=kong`
 *  - `filter[attributes.env]=dev`
 *  - `filter[attributes.env][ocontains]=dev,test`
 */
@friendlyName("AttributesFieldFilter")
@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/AttributesFieldFilter")
model AttributesFieldFilter {}
