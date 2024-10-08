import "@typespec/openapi3";

using TypeSpec.OpenAPI;

/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 */
// See: https://github.com/ulid/spec/issues/94
@pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
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
scalar Key extends string;

/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 * A key is a unique string that is used to identify a resource.
 *
 * TODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.
 */
@pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
@minLength(1)
@maxLength(64)
scalar ULIDOrKey extends string;

// NOTE (andras): key format enforcement isn't supported by TypeSpec (patternProperties). See: https://github.com/microsoft/typespec/discussions/1626
// TODO: decide if we want to use the generated Metadata type instead and update code to use it
/**
 * Set of key-value pairs.
 * Metadata can be used to store additional information about a resource.
 */
@extension("x-go-type", "map[string]string")
@example(#{ externalId: "019142cc-a016-796a-8113-1a942fecd26d" })
model Metadata {
  ...Record<string>;
}

/**
 * [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
 */
@encode(DateTimeKnownEncoding.rfc3339)
@example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
scalar DateTime extends utcDateTime;

/**
 * The monetary amount as a string to preserve decimal precision.
 */
@pattern("^-?(?:0|[1-9]\\d*)(?:\\.\\d+)?$")
@example("0.0015")
scalar Money extends string;

/**
 * Represents a resource with a unique key.
 */
@friendlyName("UniqueResource")
model UniqueResource {
  ...Resource;

  /**
   * A semi-unique identifier for the resource.
   */
  @summary("Key")
  key: Key;
}

/**
 * Represents common fields of resources.
 */
@friendlyName("Resource")
model Resource {
  /**
   * A unique identifier for the resource.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;

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
  metadata?: Metadata; // FIXME: linter error for: = #{};

  ...ResourceTimestamps;
}

/**
 * Collects the timestamps used by all resources.
 */
@friendlyName("Timestamps")
model ResourceTimestamps {
  /**
   * Timestamp of when the resource was created.
   */
  @summary("Creation Time")
  @visibility("read")
  @example(DateTime.fromISO("2024-01-01T01:01:01.001Z"))
  createdAt: DateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @summary("Last Update Time")
  @visibility("read")
  @example(DateTime.fromISO("2024-01-01T01:01:01.001Z"))
  updatedAt: DateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @summary("Deletion Time")
  @visibility("read")
  @example(DateTime.fromISO("2024-01-01T01:01:01.001Z"))
  deletedAt?: DateTime;
}

/**
 * Represents common fields of resources that can be archived.
 */
@friendlyName("Archiveable")
model Archiveable {
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
@friendlyName("RecurringPeriodInterval")
enum RecurringPeriodInterval {
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  Day: "DAY",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  Week: "WEEK",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  Month: "MONTH",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  Year: "YEAR",
}

/**
 * Recurring period with an interval and an anchor.
 */
@example(#{ interval: RecurringPeriodInterval.Day, anchor: DateTime.fromISO("2023-01-01T01:01:01.001Z") })
@friendlyName("RecurringPeriod")
model RecurringPeriod {
  /**
   * The unit of time for the interval.
   */
  @summary("Interval")
  interval: RecurringPeriodInterval;

  /**
   * A date-time anchor to base the recurring period on.
   */
  @summary("Anchor time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  anchor: DateTime;
}

/**
 * Recurring period with an interval and an anchor.
 */
@example(#{ interval: RecurringPeriodInterval.Day, anchor: DateTime.fromISO("2023-01-01T01:01:01.001Z") })
@friendlyName("RecurringPeriodCreateInput")
model RecurringPeriodCreateInput {
  /**
   * The unit of time for the interval.
   */
  @summary("Interval")
  interval: RecurringPeriodInterval;

  /**
   * A date-time anchor to base the recurring period on.
   */
  @summary("Anchor time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  anchor?: DateTime;
}

/**
 * A period with a start and end time.
 */
@friendlyName("Period")
model Period {
  /**
   * Period start time.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  from: DateTime;

  /**
   * Period end time.
   */
  @example(DateTime.fromISO("2023-02-01T01:01:01.001Z"))
  to: DateTime;
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

/**
 * Address
 */
@friendlyName("Address")
model Address {
  country?: CountryCode;
  postalCode?: string;
  state?: string;
  city?: string;
  line1?: string;
  line2?: string;
  phoneNumber?: string;
}

/**
 * Meta object to generate create/update request from type by omitting readonly properties.
 */
@friendlyName("{name}Request", T)
model Request<T, Keys extends string> {
  ...OmitProperties<T, Keys>;
}

/**
 * Set of key-value pairs managed by the system. Cannot be modified by user.
 */
@example(#{ externalId: "019142cc-a016-796a-8113-1a942fecd26d" })
@friendlyName("Annotations")
model Annotations {
  ...Record<unknown>;
}

/**
 * Numeric represents an arbitrary precision number.
 */
@extension("x-go-type", "alpacadecimal.Decimal")
@extension(
  "x-go-type-import",
  {
    path: "github.com/alpacahq/alpacadecimal",
  }
)
@pattern("^\\-?[0-9]+(\\.[0-9]+)?$")
@friendlyName("Numeric")
scalar Numeric extends string;

/**
 * Numeric representation of a percentage
 */
@friendlyName("Percentage")
@pattern("^\\-?[0-9]+(\\.[0-9]+)?%$")
// TODO: let's add some default go implementation
scalar Percentage extends Numeric;
