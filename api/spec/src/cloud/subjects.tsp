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
  @summary("List subjects")
  list(): OpenMeter.Subject[] | OpenMeter.CommonErrors;

  /**
   * Get subject by ID or key.
   */
  @get
  @operationId("getSubject")
  @summary("Get subject")
  get(
    @path subjectIdOrKey: string,
  ): OpenMeter.Subject | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Upserts a subject. Creates or updates subject.
   *
   * If the subject doesn't exist, it will be created.
   * If the subject exists, it will be partially updated with the provided fields.
   */
  @post
  @operationId("upsertSubject")
  @summary("Upsert subject")
  upsert(
    @body subject: SubjectUpsert[],
  ): OpenMeter.Subject[] | OpenMeter.CommonErrors;

  /**
   * Delete subject by ID or key.
   */
  @delete
  @operationId("deleteSubject")
  @summary("Delete subject")
  delete(@path subjectIdOrKey: string): void | OpenMeter.CommonErrors;
}

/**
 * A subject is a unique identifier for a user or entity.
 */
@friendlyName("SubjectUpsert")
@example(#{
  key: "customer-id",
  displayName: "Customer Name",
  metadata: #{ hubspotId: "123456" },
  currentPeriodStart: DateTime.fromISO("2023-01-01T00:00:00Z"),
  currentPeriodEnd: DateTime.fromISO("2023-02-01T00:00:00Z"),
  stripeCustomerId: "cus_JMOlctsKV8",
})
model SubjectUpsert is Rest.Resource.ResourceCreateModel<OpenMeter.Subject>;
