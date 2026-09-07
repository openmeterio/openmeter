import "../shared/index.tsp";

namespace Billing;

/**
 * Severity level of a billing validation issue.
 */
@friendlyName("BillingValidationIssueSeverity")
enum ValidationIssueSeverity {
  /**
   * A critical issue that prevents the resource from advancing.
   */
  Critical: "critical",

  /**
   * A warning that does not prevent the resource from advancing.
   */
  Warning: "warning",
}

/**
 * A validation issue found while processing a billing resource.
 */
@friendlyName("BillingValidationIssue")
model ValidationIssue is Shared.BaseError<string> {
  /**
   * Severity of the validation issue.
   */
  @visibility(Lifecycle.Read)
  @summary("Severity")
  severity: ValidationIssueSeverity;

  /**
   * JSON path to the field that caused the validation issue, if applicable.
   */
  @visibility(Lifecycle.Read)
  @summary("Field path")
  field?: string;

  /**
   * Component that reported the validation issue, if applicable.
   */
  @visibility(Lifecycle.Read)
  @summary("Component")
  component?: string;
}
