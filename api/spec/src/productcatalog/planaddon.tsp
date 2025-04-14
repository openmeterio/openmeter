import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 *  The AddonAssignment describes the association between a plan and add-on.
 */
@friendlyName("PlanAddon")
model PlanAddon {
  ...ResourceTimestamps;

  /**
   * Partially populated add-on properties.
   */
  @summary("Addon")
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
