import type { Decl } from "./decl.js";
import type { Service } from "./service.js";

export interface SdkModel {
  /** Go module path written into go.mod. */
  readonly module: string;
  /** Root Go package name (e.g. "openmeter"). */
  readonly packageName: string;
  /** Embedded SDKVersion. */
  readonly sdkVersion: string;
  /** Default User-Agent header. */
  readonly userAgent: string;
  /** API title from @service (falls back to packageName). */
  readonly title: string;
  /** API description from @summary on the service namespace, if present. */
  readonly summary?: string;
  /** Server base URLs declared in TypeSpec via @server. */
  readonly servers: readonly Server[];
  /** All services (one file per service). */
  readonly services: readonly Service[];
  /** All component declarations (models, enums, unions, aliases). */
  readonly components: readonly Decl[];
  /** Set of HTTP status codes for which error types must be generated. */
  readonly errorStatusCodes: readonly number[];
}

export interface Server {
  readonly url: string;
  readonly description: string;
  /** Variables in the URL template (e.g. `{port}`). */
  readonly variables: readonly ServerVariable[];
}

export interface ServerVariable {
  readonly name: string;
  readonly default?: string;
}
