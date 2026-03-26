import "./addon.tsp";
import "./plan.tsp";

namespace ProductCatalog;

/**
 * PlanAddon represents an association between a plan and an add-on,
 * controlling which add-ons are available for purchase within a plan.
 */
@friendlyName("PlanAddon")
model PlanAddon {
  ...Shared.Resource;

  /**
   * The add-on associated with the plan.
   */
  @summary("Add-on")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  addon: Shared.ResourceReference<Addon>;

  /**
   * The key of the plan phase from which the add-on becomes available for purchase.
   */
  @summary("From plan phase")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  from_plan_phase: Shared.ResourceKey;

  /**
   * The maximum number of times the add-on can be purchased for the plan.
   * Not applicable for single instance add-ons.
   */
  @summary("Max quantity")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minValue(1)
  max_quantity?: integer;

  /**
   * List of validation errors.
   */
  @summary("Validation errors")
  @visibility(Lifecycle.Read)
  validation_errors?: ProductCatalogValidationError[];
}
