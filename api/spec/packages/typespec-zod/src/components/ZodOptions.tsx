import type { Children } from "@alloy-js/core";
import { type ZodCustomEmitOptions, ZodOptionsContext } from "../context/zod-options.js";

export interface ZodOptionsProps {
  /**
   * Provide custom component for rendering a specific TypeSpec type.
   */
  customEmit: ZodCustomEmitOptions;
  children: Children;
}

/**
 * Set ZodOptions for the children of this component.
 */
export function ZodOptions(props: ZodOptionsProps) {
  return (
    <ZodOptionsContext.Provider value={{ customEmit: props.customEmit }}>{props.children}</ZodOptionsContext.Provider>
  );
}
