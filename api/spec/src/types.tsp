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
scalar DateTime extends utcDateTime;
