import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 *  The AddonAssignment describes the association between a plan and add-on.
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
   * Partially populated add-on properties.
   */
  @summary("Addon")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  addon: {
    /**
     * The ID of the add-on.
     */
    @summary("ID")
    @visibility(Lifecycle.Read, Lifecycle.Create)
    id: ULID;

    /**
     * A semi-unique identifier for the resource.
     */
    @visibility(Lifecycle.Read)
    @summary("Key")
    key: Key;

    /**
     * The version of the Add-on which templates this instance.
     */
    @summary("Version")
    @minValue(1)
    @visibility(Lifecycle.Read)
    version: integer = 1;

    /**
     * The instance type of the add-on.
     */
    @visibility(Lifecycle.Read)
    @summary("InstanceType")
    instanceType: AddonInstanceType;
  };

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
   * The add-on to create.
   */
  @summary("Addon")
  @visibility(Lifecycle.Create)
  addon: {
    /**
     * The ID of the add-on.
     */
    id: ULID;
  };
}
