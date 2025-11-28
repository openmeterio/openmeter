import "@typespec/http";
import "../common/parameters.tsp";

namespace Shared;

/**
 * Resource filters.
 */
@friendlyName("ResourceFilters")
model ResourceFilters {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  name?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  labels?: Common.LabelsFieldFilter;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  public_labels?: Common.LabelsFieldFilter;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  created_at?: Common.DateTimeFieldFilter;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  updated_at?: Common.DateTimeFieldFilter;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  deleted_at?: Common.DateTimeFieldFilter;
}
