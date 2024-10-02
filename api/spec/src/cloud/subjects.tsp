import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeterCloud;

@route("/api/v1/subjects")
@tag("Subjects")
interface Subjects {
  /**
   * List subjects.
   */
  @get
  @operationId("listSubjects")
  list(): OpenMeter.Subject[] | OpenMeter.CommonErrors;

  /**
   * Get subject by ID or key.
   */
  @get
  @operationId("getSubject")
  get(@path subjectIdOrKey: string): OpenMeter.Subject | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Upserts a subject. Creates or updates subject.
   *
   * If the subject doesn't exist, it will be created.
   * If the subject exists, it will be partially updated with the provided fields.
   */
  @post
  @operationId("upsertSubject")
  upsert(@body subject: OpenMeter.Subject[]): OpenMeter.Subject[] | OpenMeter.CommonErrors;

  /**
   * Delete subject by ID or key.
   */
  @delete
  @operationId("deleteSubject")
  delete(@path subjectIdOrKey: string): void | OpenMeter.CommonErrors;
}
