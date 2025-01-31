namespace OpenMeter.Info;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/info/progress")
@tag("Track Progress")
interface Progresses {
  /**
   * Get progress
   */
  @get
  @route("/{id}")
  @operationId("getProgress")
  @summary("Get progress")
  getProgress(
    id: string,
  ): Progress | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Progress describes a progress of a task.
 */
@friendlyName("Progress")
model Progress {
  /**
   * Success is the number of items that succeeded
   */
  success: uint64;

  /**
   * Failed is the number of items that failed
   */
  failed: uint64;

  /**
   * The total number of items to process
   */
  total: uint64;

  /**
   * The time the progress was last updated
   */
  updatedAt: DateTime;
}
