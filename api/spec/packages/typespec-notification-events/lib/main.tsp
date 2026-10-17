import "./index.js";

/**
 * Decorators that drive Notification Event contract code generation: which Go
 * package owns a TypeSpec namespace, and which payload models are Event types.
 */
namespace NotificationCodegen;

/**
 * Declares the Go package generated from every model, enum and union declared
 * directly in the namespace.
 *
 * @param path Go import path of the package.
 */
extern dec goPackage(
  target: TypeSpec.Reflection.Namespace,
  path: valueof string
);

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
