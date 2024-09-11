// NOTE (andras): key format enforcement isn't supported by TypeSpec (patternProperties). See: https://github.com/microsoft/typespec/discussions/1626
/**
 * Set of key-value pairs.
 * Metadata can be used to store additional information about a resource.
 */
@example(#{ externalId: "019142cc-a016-796a-8113-1a942fecd26d" })
model Metadata {
  ...Record<string>;
}
