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
