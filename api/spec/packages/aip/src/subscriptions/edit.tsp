import "@typespec/openapi";
import "../shared/index.tsp";
import "../productcatalog/ratecard.tsp";
import "./subscription.tsp";

using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Request for editing a running subscription. Applies an ordered batch of
 * customizations to the subscription's phases and items. A later customization
 * observes the state produced by earlier ones.
 */
@friendlyName("BillingSubscriptionEdit")
model SubscriptionEdit {
  /**
   * The ordered batch of customizations to apply to the running subscription.
   */
  @maxItems(100)
  @summary("Customizations")
  customizations: SubscriptionEditOperation[];

  /**
   * When the requested changes should take effect. Defaults to immediate.
   */
  @summary("Timing")
  timing?: SubscriptionEditTiming = SubscriptionEditTimingEnum.Immediate;
}

/**
 * The type of a subscription edit operation.
 *
 * - `add_item`: Add a rate card to a phase.
 * - `remove_item`: Remove a rate card from a phase.
 * - `add_phase`: Add a new phase to the subscription.
 * - `remove_phase`: Remove a phase from the subscription.
 * - `stretch_phase`: Extend the duration of a phase.
 * - `unschedule_edit`: Discard scheduled edits on the current phase.
 */
@friendlyName("BillingSubscriptionEditOperationType")
enum SubscriptionEditOperationType {
  AddItem: "add_item",
  RemoveItem: "remove_item",
  AddPhase: "add_phase",
  RemovePhase: "remove_phase",
  StretchPhase: "stretch_phase",
  UnscheduleEdit: "unschedule_edit",
}

/**
 * A single customization to apply to a running subscription. The `type` field
 * discriminates which operation is performed.
 */
@friendlyName("BillingSubscriptionEditOperation")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union SubscriptionEditOperation {
  add_item: SubscriptionEditAddItem,
  remove_item: SubscriptionEditRemoveItem,
  add_phase: SubscriptionEditAddPhase,
  remove_phase: SubscriptionEditRemovePhase,
  stretch_phase: SubscriptionEditStretchPhase,
  unschedule_edit: SubscriptionEditUnscheduleEdit,
}

/**
 * Add a new rate card to a phase. Adding an item to the current phase closes the
 * active version of the same item key and appends a new version.
 */
@friendlyName("BillingSubscriptionEditAddItem")
model SubscriptionEditAddItem {
  /**
   * Discriminator for the add-item operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.AddItem;

  /**
   * The key of the phase to add the item to.
   */
  @summary("Phase key")
  phase_key: string;

  /**
   * The rate card describing what the customer gets and pays for the new item.
   */
  @summary("Rate card")
  rate_card: ProductCatalog.RateCard;
}

/**
 * Remove a rate card from a phase.
 */
@friendlyName("BillingSubscriptionEditRemoveItem")
model SubscriptionEditRemoveItem {
  /**
   * Discriminator for the remove-item operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.RemoveItem;

  /**
   * The key of the phase to remove the item from.
   */
  @summary("Phase key")
  phase_key: string;

  /**
   * The key of the item to remove.
   */
  @summary("Item key")
  item_key: string;
}

/**
 * Add a new phase to the subscription. The phase is created without items; use
 * add-item operations to populate it.
 */
@friendlyName("BillingSubscriptionEditAddPhase")
model SubscriptionEditAddPhase {
  /**
   * Discriminator for the add-phase operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.AddPhase;

  /**
   * The phase to add.
   */
  @summary("Phase")
  phase: SubscriptionPhaseCreate;
}

/**
 * Remove a phase from the subscription.
 */
@friendlyName("BillingSubscriptionEditRemovePhase")
model SubscriptionEditRemovePhase {
  /**
   * Discriminator for the remove-phase operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.RemovePhase;

  /**
   * The key of the phase to remove.
   */
  @summary("Phase key")
  phase_key: string;

  /**
   * The direction to shift surrounding phases to fill the removed phase's span.
   */
  @summary("Shift")
  shift: RemovePhaseShifting;
}

/**
 * Extend the duration of a phase, shifting later phases by the same amount.
 */
@friendlyName("BillingSubscriptionEditStretchPhase")
model SubscriptionEditStretchPhase {
  /**
   * Discriminator for the stretch-phase operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.StretchPhase;

  /**
   * The key of the phase to stretch.
   */
  @summary("Phase key")
  phase_key: string;

  /**
   * The ISO-8601 duration to extend the phase by.
   */
  @summary("Extend by")
  extend_by: Shared.ISO8601Duration;
}

/**
 * Discard any scheduled edits on the current phase, reverting it to its previously
 * persisted state.
 */
@friendlyName("BillingSubscriptionEditUnscheduleEdit")
model SubscriptionEditUnscheduleEdit {
  /**
   * Discriminator for the unschedule-edit operation.
   */
  @summary("Type")
  type: SubscriptionEditOperationType.UnscheduleEdit;
}

/**
 * The direction to shift surrounding phases when a phase is removed.
 *
 * - `next`: Shift all subsequent phases to start sooner by the removed phase's
 *   length.
 * - `prev`: Extend the previous phase to end later by the removed phase's length.
 */
@friendlyName("BillingSubscriptionRemovePhaseShifting")
enum RemovePhaseShifting {
  Next: "next",
  Prev: "prev",
}

/**
 * The definition of a new subscription phase to add via an edit.
 */
@friendlyName("BillingSubscriptionPhaseCreate")
model SubscriptionPhaseCreate {
  /**
   * A locally unique identifier for the phase.
   */
  @summary("Key")
  key: Shared.ResourceKey;

  /**
   * The name of the phase.
   */
  @summary("Name")
  name: string;

  /**
   * An optional description of the phase.
   */
  @summary("Description")
  description?: string;

  /**
   * The ISO-8601 interval after the subscription start at which the phase begins.
   * When null, the phase starts immediately after the subscription starts.
   */
  @summary("Start after")
  start_after: Shared.ISO8601Duration | null;

  /**
   * The intended ISO-8601 duration of the phase. Required unless the phase will be
   * the last phase of the subscription.
   */
  @summary("Duration")
  duration?: Shared.ISO8601Duration;
}
