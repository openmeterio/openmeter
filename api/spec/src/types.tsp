import "@typespec/openapi3";

using TypeSpec.OpenAPI;

/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 */
// See: https://github.com/ulid/spec/issues/94
@pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
@example("01G65Z755AFWAKHE12NY0CQ9FH")
@extension("x-inline", true)
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
@extension("x-inline", true)
scalar Key extends string;

/**
 * ExternalKey is a looser version of key.
 */
@maxLength(256)
@minLength(1)
@extension("x-inline", true)
scalar ExternalKey extends string;

/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 * A key is a unique string that is used to identify a resource.
 *
 * TODO: this is a temporary solution to support both ULID and Key in the same spec for codegen.
 */
@pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$|^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
@minLength(1)
@maxLength(64)
@extension("x-inline", true)
scalar ULIDOrKey extends string;

/**
 * ULID (Universally Unique Lexicographically Sortable Identifier) or external unique key.
 */
@extension("x-go-type", "string")
@friendlyName("ULIDOrExternalKey")
union ULIDOrExternalKey {
  id: ULID,
  key: ExternalKey,
}

// NOTE (andras): key format enforcement isn't supported by TypeSpec (patternProperties). See: https://github.com/microsoft/typespec/discussions/1626
// TODO: decide if we want to use the generated Metadata type instead and update code to use it
/**
 * Set of key-value pairs.
 * Metadata can be used to store additional information about a resource.
 */
@extension("x-go-type", "map[string]string")
@example(#{ externalId: "019142cc-a016-796a-8113-1a942fecd26d" })
@friendlyName("Metadata")
model Metadata {
  ...Record<string>;
}

/**
 * [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in UTC.
 */
@encode(DateTimeKnownEncoding.rfc3339)
@example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
@extension("x-inline", true)
scalar DateTime extends utcDateTime;

/**
 * Represents a resource with a key.
 */
@friendlyName("KeyedResource")
model Keyed {
  /**
   * A locally unique identifier for the resource.
   */
  key: Key;
}

/**
 * Represents a resource with a unique key.
 */
@friendlyName("UniqueResource")
model UniqueResource {
  ...Resource;

  /**
   * A semi-unique identifier for the resource.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Key")
  key: Key;
}

/**
 * IDResource is a resouce with an ID.
 */
// NOTE: this can be used to have a type, that we can later replace with the expanded type if needed without
// breaking api compatibility
@friendlyName("IDResource")
model IDResource {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;
}

/**
 * Represents common fields of resources.
 */
@friendlyName("Resource")
model Resource {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
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
  metadata?: Metadata | null;

  ...ResourceTimestamps;
}

/**
 * Represents resources that can be cadenced, have scheduled activity changes.
 */
@friendlyName("CadencedResource")
model CadencedResource {
  /**
   * The cadence start of the resource.
   */
  activeFrom: DateTime;

  /**
   * The cadence end of the resource.
   */
  activeTo?: DateTime;
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
  @visibility(Lifecycle.Read)
  @example(DateTime.fromISO("2024-01-01T01:01:01.001Z"))
  createdAt: DateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @summary("Last Update Time")
  @visibility(Lifecycle.Read)
  @example(DateTime.fromISO("2024-01-01T01:01:01.001Z"))
  updatedAt: DateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @summary("Deletion Time")
  @visibility(Lifecycle.Read)
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
  @visibility(Lifecycle.Read)
  archivedAt?: DateTime;
}

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
@pattern("^P(?:\\d+(?:\\.\\d+)?Y)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?W)?(?:\\d+(?:\\.\\d+)?D)?(?:T(?:\\d+(?:\\.\\d+)?H)?(?:\\d+(?:\\.\\d+)?M)?(?:\\d+(?:\\.\\d+)?S)?)?$")
@extension("x-inline", true)
scalar ISO8601Duration extends string;

/**
 * Period duration for the recurrence
 */
@friendlyName("RecurringPeriodInterval")
union RecurringPeriodInterval {
  ISO8601Duration,
  RecurringPeriodIntervalEnum,
}

/**
 * The unit of time for the interval.
 * One of: `day`, `week`, `month`, or `year`.
 */
@friendlyName("RecurringPeriodIntervalEnum")
enum RecurringPeriodIntervalEnum {
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
#deprecated "Use RecurringPeriodV2 instead"
@example(#{
  interval: RecurringPeriodIntervalEnum.Day,
  intervalISO: duration.fromISO("P1D"),
  anchor: DateTime.fromISO("2023-01-01T01:01:01.001Z"),
})
@friendlyName("RecurringPeriod")
model RecurringPeriod {
  ...RecurringPeriodV2;

  /**
   * The unit of time for the interval in ISO8601 format.
   */
  @encode(DurationKnownEncoding.ISO8601)
  intervalISO: duration;
}

/**
 * Recurring period with an interval and an anchor.
 */
@friendlyName("RecurringPeriodV2")
model RecurringPeriodV2 {
  /**
   * The unit of time for the interval. Heuristically maps ISO duraitons to enum values or returns the ISO duration.
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
@example(#{
  interval: RecurringPeriodIntervalEnum.Day,
  anchor: DateTime.fromISO("2023-01-01T01:01:01.001Z"),
})
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
// TODO: add helpers for currency database
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
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.
   */
  country?: CountryCode;

  /**
   * Postal code.
   */
  postalCode?: string;

  /**
   * State or province.
   */
  state?: string;

  /**
   * City.
   */
  city?: string;

  /**
   * First line of the address.
   */
  line1?: string;

  /**
   * Second line of the address.
   */
  line2?: string;

  /**
   * Phone number.
   */
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
@pattern("^\\-?[0-9]+(\\.[0-9]+)?$")
@friendlyName("Numeric")
scalar Numeric extends string;
alias Money = Numeric;

/**
 * Numeric representation of a percentage
 *
 * 50% is represented as 50
 */
@example(50)
@friendlyName("Percentage")
@extension("x-go-type", "models.Percentage")
@extension("x-go-package", "github.com/openmeterio/openmeter/pkg/models")
scalar Percentage extends float64;

/**
 * Unit describes how the quantity of the product should be interpreted.
 */
@friendlyName("Unit")
scalar Unit extends string;

/**
 * TaxIdentificationCode is a normalized tax code shown on the original identity document.
 */
@minLength(1)
@maxLength(32)
@friendlyName("BillingTaxIdentificationCode")
scalar TaxIdentificationCode extends string;
