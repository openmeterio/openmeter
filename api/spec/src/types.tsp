// See: https://github.com/ulid/spec/issues/94
@pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
@friendlyName("ULID")
scalar ULID extends string;
