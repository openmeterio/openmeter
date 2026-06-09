import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

namespace Events;

/**
 * Subject of an event.
 */
@friendlyName("MeteringEventSubject")
model MeteringEventSubject {
  /**
   * The key of the subject.
   */
  @minLength(1)
  @visibility(Lifecycle.Read)
  @summary("Key")
  @example("customer-id")
  key: string;
}
