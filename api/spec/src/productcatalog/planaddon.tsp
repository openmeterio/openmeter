import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter;

/**
 *  The PlanAddon describes the association between a plan and add-on.
 */
@friendlyName("PlanAddon")
model PlanAddon {
  ...ResourceTimestamps;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Annotations;

  /**
   * Additional metadata for the resource.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Metadata")
  metadata?: Metadata;

  /**
   * Add-on object.
   */
  @summary("Addon")
  @visibility(Lifecycle.Read)
  addon: Addon;

  /**
   * The key of the plan phase from the add-on becomes available for purchase.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("The plan phase from the add-on becomes purchasable")
  fromPlanPhase: string;

  /**
   * The maximum number of times the add-on can be purchased for the plan.
   * It is not applicable for add-ons with single instance type.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Max quantity of the add-on")
  maxQuantity?: integer;

  /**
   * List of validation errors.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validationErrors: ValidationError[] | null;
}

/**
 * Order by options for plan add-on assignments.
 */
@friendlyName("PlanAddonOrderBy")
enum PlanAddonOrderBy {
  id: "id",
  key: "key",
  version: "version",
  createdAt: "created_at",
  updatedAt: "updated_at",
}

/**
 * A plan add-on assignment create request.
 */
@friendlyName("PlanAddonCreate")
@withVisibility(Lifecycle.Create)
model PlanAddonCreate {
  ...OmitProperties<DefaultKeyVisibility<PlanAddon, Lifecycle.Read>, "addon">;

  /**
   * The add-on unique identifier in ULID format.
   */
  @summary("Add-on unique identifier")
  @visibility(Lifecycle.Create)
  addonId: ULID;
}
