import "./index.js";

/**
 * Decorators that drive Notification Event contract code generation: which payload
 * models are Event types.
 */
namespace NotificationCodegen;

/**
 * Declares the model as the payload of a canonical Notification Event type.
 *
 * @param eventType Canonical Event type, `v<version>.<name>`, e.g.
 * `v1.entitlement.balance_threshold.triggered`.
 * @param ruleConfig Rule configuration model; omitted for system Event types
 * that no Rule can produce.
 */
extern dec eventType(
  target: TypeSpec.Reflection.Model,
  eventType: valueof string,
  ruleConfig?: TypeSpec.Reflection.Model
);
