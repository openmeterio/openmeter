import type { GoType } from "./type.js";

export type Param = PathParam | QueryParam | HeaderParam;

export interface PathParam {
  readonly kind: "path";
  /** Go field name (PascalCase). */
  readonly name: string;
  /** Path template name (e.g. "customerId"). */
  readonly wireName: string;
  readonly type: GoType;
  readonly doc?: string;
}

export interface QueryParam {
  readonly kind: "query";
  readonly name: string;
  readonly wireName: string;
  readonly type: GoType;
  readonly optional: boolean;
  /**
   * deepObject => `?filter[id]=...` (used for filter / page params)
   * form       => `?key=value` (default)
   */
  readonly style: "form" | "deepObject";
  readonly explode: boolean;
  readonly doc?: string;
}

export interface HeaderParam {
  readonly kind: "header";
  readonly name: string;
  readonly wireName: string;
  readonly type: GoType;
  readonly optional: boolean;
  readonly doc?: string;
}
