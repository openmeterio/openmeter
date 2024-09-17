/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 */
// See: https://github.com/ulid/spec/issues/94
@pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
@friendlyName("ULID")
@example("01G65Z755AFWAKHE12NY0CQ9FH")
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
@friendlyName("Key")
scalar Key extends string;

// NOTE (andras): key format enforcement isn't supported by TypeSpec (patternProperties). See: https://github.com/microsoft/typespec/discussions/1626
/**
 * Set of key-value pairs.
 * Metadata can be used to store additional information about a resource.
 */
@example(#{ externalId: "019142cc-a016-796a-8113-1a942fecd26d" })
@friendlyName("Metadata")
model Metadata {
  ...Record<string>;
}

/**
 * [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
 */
@encode("rfc3339")
@example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
scalar DateTime extends utcDateTime;

/**
 * The monetary amount as a string to preserve decimal precision.
 */
@pattern("^-?(?:0|[1-9]\\d*)(?:\\.\\d+)?$")
@example("0.0015")
scalar Money extends string;

/**
 * Represents common fields of resources.
 */
@friendlyName("Resource")
model Resource {
  /**
   * A semi-unique identifier for the resource.
   */
  @summary("Key")
  key: Key;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @summary("Display name")
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024)
  @summary("Description")
  description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata")
  metadata?: Metadata = #{};

  /**
   * Timestamp of when the resource was created.
   */
  @summary("Creation Time")
  @visibility("read")
  createdAt?: DateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @summary("Last Update Time")
  @visibility("read")
  updatedAt?: DateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @summary("Deletion Time")
  @visibility("read")
  deletedAt?: DateTime;

  /**
   * Timestamp of when the resource was archived.
   */
  @summary("Archival Time")
  @visibility("read")
  archivedAt?: DateTime;
}

/**
 * The unit of time for the interval.
 * One of: `day`, `week`, `month`, or `year`.
 */
@friendlyName("IntervalUnit")
union IntervalUnit {
  @summary("Day")
  day: "day",

  @summary("Week")
  week: "week",

  @summary("Month")
  month: "month",

  @summary("Year")
  year: "year",
}

/**
 * Time interval with a time unit and count.
 */
@example(#{ unit: "month", count: 3 })
@friendlyName("Interval")
model Interval {
  /**
   * The unit of time for the interval.
   */
  @summary("Interval unit")
  unit: IntervalUnit;

  /**
   * The number of time units in the interval.
   */
  @minValue(1)
  @summary("Interval count")
  count?: integer = 1;
}

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
