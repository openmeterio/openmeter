import type { Operation } from "./operation.js";

export interface Service {
  /** Service name (e.g. "Customers"). */
  readonly name: string;
  /** Generated file name (e.g. "openmetercustomers.go"). */
  readonly fileName: string;
  /** Field name on the root SDK struct (e.g. "OpenMeterCustomers"). */
  readonly rootFieldName: string;
  /** Go struct name (e.g. "OpenMeterCustomers"). */
  readonly structName: string;
  /** Constructor name (e.g. "newOpenMeterCustomers"). */
  readonly ctorName: string;
  readonly operations: readonly Operation[];
}
