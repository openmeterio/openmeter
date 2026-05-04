import type { GoType } from "./type.js";
import type { Param } from "./parameter.js";

/** An HTTP operation. */
export interface Operation {
  /** TypeSpec @operationId (kebab-case). */
  readonly id: string;
  /** Method name on the Go service struct (PascalCase). */
  readonly methodName: string;
  /** Service this operation belongs to (e.g. "Customers"). */
  readonly service: string;
  readonly verb: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS";
  /** URI template path (e.g. "/openmeter/customers/{customerId}"). */
  readonly path: string;
  readonly params: readonly Param[];
  readonly body?: OperationBody;
  readonly responses: readonly OperationResponse[];
  readonly doc?: string;
}

export interface OperationBody {
  /** Go field name on the request struct (e.g. "Customer"). */
  readonly name: string;
  readonly type: GoType;
  readonly contentType: string;
}

export interface OperationResponse {
  /** HTTP status code (e.g. 200, 400). */
  readonly status: number;
  /** Response body type. Undefined for 204 / empty responses. */
  readonly bodyType?: GoType;
  /** Content type (e.g. "application/json"). Undefined for empty responses. */
  readonly contentType?: string;
  /** Whether this is an error response (status >= 400). */
  readonly isError: boolean;
  /**
   * For error responses, the name of the apierrors type to use (e.g. "BadRequestError").
   * Generated separately from `models/apierrors/`.
   */
  readonly errorTypeName?: string;
}
