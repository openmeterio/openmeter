import "../common/properties.tsp";

namespace Shared;

// See: https://github.com/ulid/spec/issues/94
/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 */
@pattern("^[0-7][0-9A-HJKMNP-TV-Z]{25}$")
@example("01G65Z755AFWAKHE12NY0CQ9FH")
@friendlyName("ULID")
@summary("ULID")
scalar ULID extends string;

/**
 * A key is a unique string that is used to identify a resource.
 */
@pattern(
  "^[a-z0-9]+(?:_[a-z0-9]+)*$",
  "Must start with a lowercase letter or a number. Can contain lowercase letters, numbers, and underscores."
)
@minLength(1)
@maxLength(64)
@friendlyName("ResourceKey")
@summary("Resource Key")
@example("resource_key")
scalar ResourceKey extends string;

/**
 * ULID ID or Resource Key.
 */
@friendlyName("ULIDOrResourceKey")
@summary("ULID ID or Resource Key")
union ULIDOrResourceKey {
  id: ULID,
  key: ResourceKey,
}

/**
 * ExternalResourceKey is a unique string that is used to identify a resource in an external system.
 */
@maxLength(256)
@minLength(1)
@friendlyName("ExternalResourceKey")
@summary("External Resource Key")
@example("019ae40f-4258-7f15-9491-842f42a7d6ac")
scalar ExternalResourceKey extends string;

/**
 * ULID ID or External Resource Key.
 */
@friendlyName("ULIDOrExternalResourceKey")
@summary("ULID ID or External Resource Key")
union ULIDOrExternalResourceKey {
  id: ULID,
  key: ExternalResourceKey,
}

/**
 * [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
 */
@encode(DateTimeKnownEncoding.rfc3339)
@friendlyName("DateTime")
@summary("RFC3339 Date-Time")
@example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
scalar DateTime extends utcDateTime;

// This is a reasonably good RE2-safe ISO8601 duration pattern that allows fractional components (though overly permissive)
// It matches patterns such as
// - P1Y
// - P1.5Y
// - P1Y2M1D
// - P1Y2MT <= invalid: must have time component defined after T
// - P1Y2MT1H2S
// - P1Y2.3MT1.4H2S <= invalid: spec only allows for a single fractional component
//
// A stricter alternative that doesn't allow fractional components is: ^P(?:\d+Y)?(?:\d+M)?(?:\d+W)?(?:\d+D)?(?:T(?:\d+H)?(?:\d+M)?(?:\d+S)?)?$
// ^before using the stricter pattern make sure to double escape the backslashes
/**
 * [ISO 8601 Duration](https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm) string.
 */
@encode(DurationKnownEncoding.ISO8601)
@pattern("^P(?:\\d+(?:\\.\\d+)?Y)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?W)?(?:\\d+(?:\\.\\d+)?D)?(?:T(?:\\d+(?:\\.\\d+)?H)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?S)?)?$")
@friendlyName("ISO8601Duration")
@summary("ISO 8601 Duration")
@example("P1Y")
scalar ISO8601Duration extends string;

/**
 * Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
 * Custom three-letter currency codes are also supported for convenience.
 */
@pattern("^[A-Z]{3}$")
@friendlyName("CurrencyCode")
@minLength(3)
@maxLength(3)
@example("USD")
scalar CurrencyCode extends string;

/**
 * [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
 * Custom two-letter country codes are also supported for convenience.
 */
@pattern("^[A-Z]{2}$")
@friendlyName("CountryCode")
@minLength(2)
@maxLength(2)
@example("US")
scalar CountryCode extends string;

/**
 * Numeric represents an arbitrary precision number.
 */
@pattern("^\\-?[0-9]+(\\.[0-9]+)?$")
@friendlyName("Numeric")
scalar Numeric extends string;

/**
 * Money represents a currency amount.
 */
alias Money = Numeric;
